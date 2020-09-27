## flatten.py

# read input file
import json
with open(r'rocketchat_message_sample.json') as f:
    dt = json.load(f)

# faltten dictionaty
from flatten_json import flatten
dic_flattened = flatten(dt)
#dic_flattened = (flatten(d) for d in dt)

# use pandas
import pandas as pd
df = pd.DataFrame(dic_flattened, index=[0])
print('-- DF COLUMNS:\n',df.columns)

# create file db
from sqlalchemy import create_engine
engine = create_engine('sqlite:///db.sqlite.flat', echo=False)

# create schema and write records to db
df.to_sql('rocketchat_message', 
        con=engine, 
        if_exists='replace',
        index=False)
print('-- DB SELECT:\n', engine.execute('SELECT * FROM rocketchat_message').fetchall())
