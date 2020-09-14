# read input file
import json
with open(r'rocketchat_message_sample.json') as f:
    dt = json.load(f)

# normalize data
import pandas as pd
from pandas.io.json import json_normalize
df = pd.DataFrame(json_normalize(dt))
print('-- DF COLUMNS:\n',df.columns)

# create file db
from sqlalchemy import create_engine
engine = create_engine('sqlite:///db.sqlite.norm', echo=False)

# create schema and write records to db
df.to_sql('rocketchat_message', 
        con=engine, 
        if_exists='replace',
        index=False)
print('-- DB SELECT:\n', engine.execute('SELECT * FROM rocketchat_message').fetchall())
