CREATE TABLE campaigns (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  name VARCHAR(255) NOT NULL,
  ad_platform VARCHAR(32) NOT NULL,
  external_campaign_id VARCHAR(128) NOT NULL,
  attribution_window_days INT NOT NULL DEFAULT 7,
  is_active BOOLEAN NOT NULL DEFAULT TRUE,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_campaign_external_id (external_campaign_id)
) ENGINE=InnoDB;

CREATE TABLE events (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  event_name VARCHAR(128) NOT NULL,
  user_id VARCHAR(128) NOT NULL,
  device_id VARCHAR(128) NOT NULL DEFAULT '',
  click_id VARCHAR(255) NOT NULL DEFAULT '',
  ad_platform VARCHAR(32) NOT NULL DEFAULT '',
  campaign_id BIGINT NOT NULL DEFAULT 0,
  properties JSON,
  occurred_at TIMESTAMP NOT NULL,
  received_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  INDEX idx_events_click_id (click_id),
  INDEX idx_events_campaign (campaign_id),
  INDEX idx_events_user_name_time (user_id, event_name, occurred_at),
  INDEX idx_events_received_at (received_at)
) ENGINE=InnoDB;

CREATE TABLE conversions (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  event_id BIGINT NOT NULL,
  ad_platform VARCHAR(32) NOT NULL,
  campaign_id BIGINT NOT NULL,
  click_id VARCHAR(255) NOT NULL DEFAULT '',
  status VARCHAR(16) NOT NULL DEFAULT 'pending',
  attempt_count INT NOT NULL DEFAULT 0,
  last_error TEXT,
  callback_payload JSON,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP ON UPDATE CURRENT_TIMESTAMP,
  INDEX idx_conversions_status (status),
  INDEX idx_conversions_campaign (campaign_id),
  CONSTRAINT fk_conversions_event FOREIGN KEY (event_id) REFERENCES events(id)
) ENGINE=InnoDB;

CREATE TABLE experiments (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  exp_key VARCHAR(128) NOT NULL,
  name VARCHAR(255) NOT NULL,
  status VARCHAR(16) NOT NULL DEFAULT 'draft',
  traffic_allocation INT NOT NULL DEFAULT 100,
  created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_experiment_key (exp_key)
) ENGINE=InnoDB;

CREATE TABLE experiment_variants (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  experiment_id BIGINT NOT NULL,
  variant_key VARCHAR(128) NOT NULL,
  name VARCHAR(255) NOT NULL,
  weight INT NOT NULL DEFAULT 1,
  CONSTRAINT fk_variants_experiment FOREIGN KEY (experiment_id) REFERENCES experiments(id),
  UNIQUE KEY uniq_experiment_variant (experiment_id, variant_key)
) ENGINE=InnoDB;

CREATE TABLE experiment_assignments (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  experiment_id BIGINT NOT NULL,
  variant_id BIGINT NOT NULL,
  user_id VARCHAR(128) NOT NULL,
  assigned_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
  UNIQUE KEY uniq_experiment_user (experiment_id, user_id),
  CONSTRAINT fk_assignments_experiment FOREIGN KEY (experiment_id) REFERENCES experiments(id),
  CONSTRAINT fk_assignments_variant FOREIGN KEY (variant_id) REFERENCES experiment_variants(id)
) ENGINE=InnoDB;

CREATE TABLE daily_funnel_stats (
  id BIGINT AUTO_INCREMENT PRIMARY KEY,
  stat_date DATE NOT NULL,
  campaign_id BIGINT NOT NULL,
  event_name VARCHAR(128) NOT NULL,
  event_count BIGINT NOT NULL DEFAULT 0,
  UNIQUE KEY uniq_daily_stat (stat_date, campaign_id, event_name)
) ENGINE=InnoDB;
