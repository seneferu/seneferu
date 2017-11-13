CREATE TABLE steps (
  uid         SERIAL PRIMARY KEY,
  buildnumber INTEGER,
  repo        VARCHAR,
  name        VARCHAR,
  log         TEXT,
  status      VARCHAR,
  exitcode    INTEGER,
  --CONSTRAINT FK_buildnumber FOREIGN KEY (buildnumber)
  --REFERENCES builds (uid),

  CONSTRAINT FK_repos FOREIGN KEY (repo)
  REFERENCES repositories (id),

  CONSTRAINT step_uq
  UNIQUE (buildnumber, repo, name)
);