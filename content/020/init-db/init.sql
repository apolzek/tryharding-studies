CREATE TABLE IF NOT EXISTS customers (
    id           BIGSERIAL PRIMARY KEY,
    customer_id  VARCHAR(50)  UNIQUE NOT NULL,
    name         VARCHAR(100) NOT NULL,
    credit_score INTEGER      NOT NULL CHECK (credit_score BETWEEN 300 AND 850),
    account_age_days INTEGER  NOT NULL,
    monthly_income   NUMERIC(12,2) NOT NULL,
    created_at   TIMESTAMP DEFAULT NOW()
);

INSERT INTO customers (customer_id, name, credit_score, account_age_days, monthly_income) VALUES
    ('CUST-001', 'Alice Smith',    750, 1825, 8500.00),
    ('CUST-002', 'Bob Johnson',    620,  365, 4200.00),
    ('CUST-003', 'Carol Williams', 580,   90, 3100.00),
    ('CUST-004', 'David Brown',    800, 3650, 12000.00),
    ('CUST-005', 'Eve Martinez',   490,   30,  1800.00)
ON CONFLICT (customer_id) DO NOTHING;

CREATE TABLE IF NOT EXISTS loan_applications (
    id                BIGSERIAL PRIMARY KEY,
    application_id    VARCHAR(100) UNIQUE NOT NULL,
    customer_id       VARCHAR(50)  NOT NULL,
    amount            NUMERIC(12,2) NOT NULL,
    currency          VARCHAR(10)  NOT NULL,
    compliance_status VARCHAR(20),
    fraud_decision    VARCHAR(20),
    created_at        TIMESTAMP DEFAULT NOW()
);
