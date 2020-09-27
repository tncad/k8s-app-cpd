## denormalization.py

# read input file
import json
fact_name = 'message'
with open(r'rocketchat_' + fact_name + '_sample.json') as f:
    data = json.load(f)

# custom: flag metrics and dimension columns
data['units'] = 0.0
data['dim_user'] = data.pop('u')
data['dim_time'] = {'_id':0,'dt':data.pop('ts')}

# custom: enrich dimensions
import pandas as pd, dateutil.parser
dt = dateutil.parser.parse(data['dim_time']['dt'])
data['dim_time']['hour'] = dt.hour
data['dim_time']['day'] = dt.day
data['dim_time']['month'] = dt.month
data['dim_time']['year'] = dt.year
data['dim_time']['weekday'] = dt.weekday()
data['dim_time']['weeknum'] = dt.isocalendar()[1]

# normalize data
from pandas.io.json import json_normalize
df = pd.DataFrame(json_normalize(data))

# create file db
from sqlalchemy import create_engine
db_uri = 'sqlite:///db.sqlite.denorm'
engine = create_engine(db_uri, echo=False)

# create temporary dimension tales
for col in df.columns:
    if col.startswith('dim_'):
        tbl = col.split('.')[0]
        pd.DataFrame(json_normalize(data[tbl])).to_sql(tbl, 
            con=engine,
            if_exists='replace',
            index=False)

# create temporary fact table
df.drop(list(df.filter(regex='dim_[^.]*\.[^_]')), axis=1, inplace=True)
df.to_sql(fact_name,
        con=engine,
        if_exists='replace',
        index=False)
