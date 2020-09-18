--  dim_time  sample data:  [(0, '2020-08-30T15:21:37.974Z', 15, 30, 8, 2020, 6, 35)] 
CREATE TABLE dim_time (
	_id BIGINT NOT NULL, 
	dt TEXT, 
	hour BIGINT, 
	day BIGINT, 
	month BIGINT, 
	year BIGINT, 
	weekday BIGINT, 
	weeknum BIGINT, 
	CONSTRAINT dim_time_pk PRIMARY KEY (_id)
);

--  dim_user  sample data:  [('3jh6AN3eLZKRCD6E9', 'tncad')] 
CREATE TABLE dim_user (
	_id TEXT NOT NULL, 
	username TEXT, 
	CONSTRAINT dim_user_pk PRIMARY KEY (_id)
);

--  message  sample data:  [('MshE4AKDejGaiwJkv', 'uj', 'GENERAL', 'tncad', False, '2020-08-30T15:21:37.974Z', 0.0, '3jh6AN3eLZKRCD6E9', 0)] 
CREATE TABLE message (
	units FLOAT, 
	"dim_user._id" TEXT, 
	"dim_time._id" BIGINT, 
	FOREIGN KEY("dim_user._id") REFERENCES dim_user (_id), 
	FOREIGN KEY("dim_time._id") REFERENCES dim_time (_id)
);
