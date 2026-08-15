package main

import (
	"database/sql"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"

	"meshy-growth-backend/internal/config"
	"meshy-growth-backend/internal/db"
	"meshy-growth-backend/internal/httpserver"
)

type createVariantReq struct {
	Key    string `json:"key" binding:"required"`
	Name   string `json:"name" binding:"required"`
	Weight int    `json:"weight" binding:"required"`
}

type createExperimentReq struct {
	Key               string             `json:"key" binding:"required"`
	Name              string             `json:"name" binding:"required"`
	TrafficAllocation int                `json:"traffic_allocation"`
	Variants          []createVariantReq `json:"variants" binding:"required,min=2"`
}

func main() {
	cfg := config.Load()
	conn, err := db.Connect(cfg.MySQLDSN)
	if err != nil {
		log.Fatalf("mysql connect: %v", err)
	}
	defer conn.Close()

	r := gin.Default()
	r.GET("/healthz", func(c *gin.Context) { c.String(http.StatusOK, "ok") })

	r.POST("/v1/experiments", func(c *gin.Context) {
		var req createExperimentReq
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		if req.TrafficAllocation == 0 {
			req.TrafficAllocation = 100
		}
		id, err := createExperiment(conn, req)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusCreated, gin.H{"id": id, "key": req.Key})
	})

	r.GET("/v1/experiments", func(c *gin.Context) {
		list, err := listExperiments(conn)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, list)
	})

	r.GET("/v1/experiments/:key", func(c *gin.Context) {
		detail, err := getExperiment(conn, c.Param("key"))
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, detail)
	})

	r.GET("/v1/experiments/:key/stats", func(c *gin.Context) {
		goal := c.DefaultQuery("goal_event", "purchase")
		stats, err := getExperimentStats(conn, c.Param("key"), goal)
		if err != nil {
			c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, stats)
	})

	log.Printf("experiment-service listening on %s", cfg.HTTPAddr)
	if err := httpserver.Run(cfg.HTTPAddr, r); err != nil {
		log.Fatal(err)
	}
}

func createExperiment(conn *sql.DB, req createExperimentReq) (int64, error) {
	tx, err := conn.Begin()
	if err != nil {
		return 0, err
	}
	res, err := tx.Exec(
		"INSERT INTO experiments (exp_key, name, status, traffic_allocation) VALUES (?, ?, 'running', ?)",
		req.Key, req.Name, req.TrafficAllocation)
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	id, err := res.LastInsertId()
	if err != nil {
		tx.Rollback()
		return 0, err
	}
	for _, v := range req.Variants {
		if _, err := tx.Exec(
			"INSERT INTO experiment_variants (experiment_id, variant_key, name, weight) VALUES (?, ?, ?, ?)",
			id, v.Key, v.Name, v.Weight); err != nil {
			tx.Rollback()
			return 0, err
		}
	}
	return id, tx.Commit()
}

type experimentSummary struct {
	ID                int64  `json:"id"`
	Key               string `json:"key"`
	Name              string `json:"name"`
	Status            string `json:"status"`
	TrafficAllocation int    `json:"traffic_allocation"`
}

func listExperiments(conn *sql.DB) ([]experimentSummary, error) {
	rows, err := conn.Query("SELECT id, exp_key, name, status, traffic_allocation FROM experiments ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	out := []experimentSummary{}
	for rows.Next() {
		var e experimentSummary
		if err := rows.Scan(&e.ID, &e.Key, &e.Name, &e.Status, &e.TrafficAllocation); err != nil {
			return nil, err
		}
		out = append(out, e)
	}
	return out, nil
}

type variantInfo struct {
	Key    string `json:"key"`
	Name   string `json:"name"`
	Weight int    `json:"weight"`
}

type experimentDetail struct {
	experimentSummary
	Variants []variantInfo `json:"variants"`
}

func getExperiment(conn *sql.DB, key string) (*experimentDetail, error) {
	var d experimentDetail
	if err := conn.QueryRow(
		"SELECT id, exp_key, name, status, traffic_allocation FROM experiments WHERE exp_key = ?", key,
	).Scan(&d.ID, &d.Key, &d.Name, &d.Status, &d.TrafficAllocation); err != nil {
		return nil, err
	}
	rows, err := conn.Query("SELECT variant_key, name, weight FROM experiment_variants WHERE experiment_id = ?", d.ID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	d.Variants = []variantInfo{}
	for rows.Next() {
		var v variantInfo
		if err := rows.Scan(&v.Key, &v.Name, &v.Weight); err != nil {
			return nil, err
		}
		d.Variants = append(d.Variants, v)
	}
	return &d, nil
}

type variantStats struct {
	VariantKey         string   `json:"variant_key"`
	Assigned           int64    `json:"assigned"`
	Converted          int64    `json:"converted"`
	ConversionRate     float64  `json:"conversion_rate"`
	ConfidenceInterval [2]float64 `json:"confidence_interval"`
	PValueVsControl    *float64 `json:"p_value_vs_control,omitempty"`
}

type srmCheck struct {
	ChiSquare float64 `json:"chi_square"`
	PValue    float64 `json:"p_value"`
	Flagged   bool    `json:"flagged"`
}

type experimentStatsResponse struct {
	Variants []variantStats `json:"variants"`
	SRMCheck *srmCheck      `json:"srm_check,omitempty"`
}

func getExperimentStats(conn *sql.DB, key, goalEvent string) (*experimentStatsResponse, error) {
	var experimentID int64
	if err := conn.QueryRow("SELECT id FROM experiments WHERE exp_key = ?", key).Scan(&experimentID); err != nil {
		return nil, err
	}
	rows, err := conn.Query(`
		SELECT ev.variant_key, ev.weight,
			COUNT(DISTINCT ea.user_id) AS assigned,
			COUNT(DISTINCT CASE WHEN e.id IS NOT NULL THEN ea.user_id END) AS converted
		FROM experiment_variants ev
		LEFT JOIN experiment_assignments ea ON ea.variant_id = ev.id
		LEFT JOIN events e ON e.user_id = ea.user_id AND e.event_name = ? AND e.occurred_at >= ea.assigned_at
		WHERE ev.experiment_id = ?
		GROUP BY ev.id, ev.variant_key, ev.weight
		ORDER BY ev.id`, goalEvent, experimentID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	variants := []variantStats{}
	weights := []int{}
	for rows.Next() {
		var s variantStats
		var weight int
		if err := rows.Scan(&s.VariantKey, &weight, &s.Assigned, &s.Converted); err != nil {
			return nil, err
		}
		if s.Assigned > 0 {
			s.ConversionRate = float64(s.Converted) / float64(s.Assigned)
		}
		low, high := wilsonInterval(s.Converted, s.Assigned, 1.96)
		s.ConfidenceInterval = [2]float64{low, high}
		variants = append(variants, s)
		weights = append(weights, weight)
	}

	// Compare every non-control variant against the first (control) variant.
	if len(variants) > 1 {
		control := variants[0]
		for i := 1; i < len(variants); i++ {
			p := twoProportionZTest(variants[i].Converted, variants[i].Assigned, control.Converted, control.Assigned)
			variants[i].PValueVsControl = &p
		}
	}

	resp := &experimentStatsResponse{Variants: variants}

	totalWeight := 0
	totalAssigned := int64(0)
	for i, w := range weights {
		totalWeight += w
		totalAssigned += variants[i].Assigned
	}
	if totalWeight > 0 && totalAssigned > 0 {
		chiSq := 0.0
		for i, w := range weights {
			expected := float64(totalAssigned) * float64(w) / float64(totalWeight)
			if expected > 0 {
				diff := float64(variants[i].Assigned) - expected
				chiSq += diff * diff / expected
			}
		}
		df := len(weights) - 1
		pValue := chiSquarePValue(chiSq, df)
		resp.SRMCheck = &srmCheck{ChiSquare: chiSq, PValue: pValue, Flagged: pValue < 0.01}
	}

	return resp, nil
}
