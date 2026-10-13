CREATE TABLE IF NOT EXISTS titanic_passengers (
  passenger_id INT PRIMARY KEY,
  survived TINYINT NOT NULL,
  pclass TINYINT NOT NULL,
  name VARCHAR(255) NOT NULL,
  sex VARCHAR(16) NOT NULL,
  age DECIMAL(5,2) NULL,
  sibsp INT NOT NULL DEFAULT 0,
  parch INT NOT NULL DEFAULT 0,
  ticket VARCHAR(64) NOT NULL DEFAULT '',
  fare DECIMAL(10,4) NULL,
  cabin VARCHAR(32) NULL,
  embarked VARCHAR(4) NULL
) ENGINE=InnoDB;

LOAD DATA LOCAL INFILE '/tmp/titanic.csv'
INTO TABLE titanic_passengers
FIELDS TERMINATED BY ',' ENCLOSED BY '"'
LINES TERMINATED BY '\n'
IGNORE 1 LINES
(passenger_id, survived, pclass, name, sex, @age, sibsp, parch, ticket, @fare, @cabin, @embarked)
SET
  age = IF(@age = '', NULL, @age),
  fare = IF(@fare = '', NULL, @fare),
  cabin = IF(@cabin = '', NULL, @cabin),
  embarked = IF(@embarked = '', NULL, @embarked);
