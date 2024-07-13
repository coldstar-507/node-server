package db

var createNodesTable = `
  CREATE TABLE IF NOT EXISTS nodes (
    id              TEXT PRIMARY KEY,
    type            TEXT NOT NULL,
    name            TEXT NOT NULL,
    messagingTokens TEXT,
    mainDeviceId    TEXT,
    hashTree        TEXT,
    ownerId         TEXT,
    lastName        TEXT,
    isPublic        BOOL,
    mediaId         TEXT,
    children        TEXT,
    posts           TEXT,
    privates        TEXT,
    admins          TEXT,
    neuter          TEXT,
    members         TEXT,
    deviceId        TEXT,
    lastUpdate      INTEGER,

    latitude        REAL,
    longitude       REAL,
  
    location        TEXT,
    age             INTEGER,
    gender          TEXT,
    interests       TEXT
  );
`

var createPrivatesTable = `
CREATE TABLE IF NOT EXISTS privates (
  id          TEXT PRIMARY KEY,
  medias      TEXT,
  connections TEXT,
  themes      TEXT
);
`

var boostq = `
  SELECT * FROM nodes
    WHERE type = 'user'
    AND location IN ('geohash1', 'geohash2', ...)
    AND age BETWEEN $1 AND $2
    AND gender IN ('male', 'female')
    AND string_to_array(interests, ' ') @> ARRAY['kush', 'money']
`

var createTypeIndex = `
  CREATE INDEX IF NOT EXISTS type_index ON nodes (
    type
  );
`

var createBoostIndex = `
  CREATE INDEX IF NOT EXISTS boost_index ON nodes (
    location, age, gender
  );
`
