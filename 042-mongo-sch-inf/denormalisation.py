# read input file
import json
with open(r'rocketchat_message_sample.json') as f:
    dt = json.load(f)

# flag dimension columns
dt['dim_user'] = dt.pop('u')
dt['dim_time'] = {'ts':dt.pop('ts')}

# normalize data
import pandas as pd
from pandas.io.json import json_normalize
df = pd.DataFrame(json_normalize(dt))

# create in-memory db
from sqlalchemy import create_engine
engine = create_engine('sqlite://', echo=False)

# create dimensions
for col in df.columns:
    if col.startswith('dim_'):
        tbl = col.split('.')[0]
        pd.DataFrame(json_normalize(dt[tbl])).to_sql(tbl, 
            con=engine,
            if_exists='replace',
            index_label='id')
        df.drop([col], axis=1)

# create fact
df.to_sql('message',
        con=engine,
        if_exists='replace',
        index_label='id')

# select data
print('-- DB SELECT FACT:\n', engine.execute('SELECT * FROM message').fetchall())
print('-- DB SELECT DIM1:\n', engine.execute('SELECT * FROM dim_user').fetchall())
print('-- DB SELECT DIM2:\n', engine.execute('SELECT * FROM dim_time').fetchall())
