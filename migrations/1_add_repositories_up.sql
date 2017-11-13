CREATE TABLE repositories (
  uid     SERIAL PRIMARY KEY,
  id      VARCHAR(300) UNIQUE,
  name    VARCHAR(300),
  url     VARCHAR(300),
  created TIMESTAMP DEFAULT NOW()
);