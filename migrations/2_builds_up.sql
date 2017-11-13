CREATE TABLE builds (
  uid       SERIAL PRIMARY KEY,
  repo      VARCHAR,
  number    INTEGER,
  comitters VARCHAR,
  created   TIMESTAMP DEFAULT NOW(),
  success   BOOLEAN,
  status    VARCHAR,
  owner     VARCHAR,
  commit    VARCHAR,
  coverage  VARCHAR,
  duration  VARCHAR,
  CONSTRAINT FK_repos FOREIGN KEY (repo)
  REFERENCES repositories (id),
  CONSTRAINT build_uq
  UNIQUE (repo, number)
);
