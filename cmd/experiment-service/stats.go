package main

import "math"

// normalCDF is the standard normal cumulative distribution function.
func normalCDF(z float64) float64 {
	return 0.5 * (1 + math.Erf(z/math.Sqrt2))
}

// twoSidedPValue converts a z-score into a two-sided p-value.
func twoSidedPValue(z float64) float64 {
	return 2 * (1 - normalCDF(math.Abs(z)))
}

// wilsonInterval computes the Wilson score confidence interval for a
// binomial proportion. It's more robust than the normal approximation when
// the sample size or the observed rate is small, which is common early in an
// experiment. z=1.96 gives a 95% interval.
func wilsonInterval(successes, total int64, z float64) (low, high float64) {
	if total == 0 {
		return 0, 0
	}
	n := float64(total)
	phat := float64(successes) / n
	denom := 1 + z*z/n
	center := phat + z*z/(2*n)
	margin := z * math.Sqrt(phat*(1-phat)/n+z*z/(4*n*n))
	low = math.Max(0, (center-margin)/denom)
	high = math.Min(1, (center+margin)/denom)
	return low, high
}

// twoProportionZTest returns the two-sided p-value for the null hypothesis
// that the conversion rate of (x1, n1) equals that of (x2, n2), using a
// pooled-variance z-test.
func twoProportionZTest(x1, n1, x2, n2 int64) float64 {
	if n1 == 0 || n2 == 0 {
		return 1
	}
	p1 := float64(x1) / float64(n1)
	p2 := float64(x2) / float64(n2)
	pPool := float64(x1+x2) / float64(n1+n2)
	se := math.Sqrt(pPool * (1 - pPool) * (1/float64(n1) + 1/float64(n2)))
	if se == 0 {
		return 1
	}
	z := (p1 - p2) / se
	return twoSidedPValue(z)
}

// chiSquarePValue returns the upper-tail p-value of a chi-square statistic
// with df degrees of freedom, i.e. the regularized upper incomplete gamma
// function Q(df/2, chiSq/2).
func chiSquarePValue(chiSq float64, df int) float64 {
	if df <= 0 {
		return 1
	}
	a := float64(df) / 2
	x := chiSq / 2
	if x <= 0 {
		return 1
	}
	if x < a+1 {
		return 1 - gammaSeries(a, x)
	}
	return gammaContinuedFraction(a, x)
}

// gammaSeries computes the regularized lower incomplete gamma function P(a,x)
// via its series expansion, valid for x < a+1.
func gammaSeries(a, x float64) float64 {
	gln, _ := math.Lgamma(a)
	ap := a
	sum := 1.0 / a
	del := sum
	for i := 0; i < 200; i++ {
		ap++
		del *= x / ap
		sum += del
		if math.Abs(del) < math.Abs(sum)*3e-16 {
			break
		}
	}
	return sum * math.Exp(-x+a*math.Log(x)-gln)
}

// gammaContinuedFraction computes the regularized upper incomplete gamma
// function Q(a,x) via its continued-fraction expansion, valid for x >= a+1.
func gammaContinuedFraction(a, x float64) float64 {
	const fpmin = 1e-300
	gln, _ := math.Lgamma(a)
	b := x + 1 - a
	c := 1 / fpmin
	d := 1 / b
	h := d
	for i := 1; i < 200; i++ {
		an := -float64(i) * (float64(i) - a)
		b += 2
		d = an*d + b
		if math.Abs(d) < fpmin {
			d = fpmin
		}
		c = b + an/c
		if math.Abs(c) < fpmin {
			c = fpmin
		}
		d = 1 / d
		del := d * c
		h *= del
		if math.Abs(del-1) < 3e-16 {
			break
		}
	}
	return math.Exp(-x+a*math.Log(x)-gln) * h
}
