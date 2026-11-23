DROP TABLE IF EXISTS instruments;migrate -path migrations \
  -database "postgres://priyanshu@localhost:5432/tradex?sslmode=disable" versionINSERT INTO instruments (symbol, company_name, exchange)
VALUES
    ('RELIANCE', 'Reliance Industries', 'NSE'),
    ('TCS', 'Tata Consultancy Services', 'NSE'),
    ('INFY', 'Infosys Limited', 'NSE'),
    ('JIOFIN', 'Jio Financial Services', 'NSE')
ON CONFLICT (exchange, symbol) DO NOTHING;