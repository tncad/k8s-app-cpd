## reflection.py

fact_name = 'message'

# open file db
from sqlalchemy import create_engine
db_uri = 'sqlite:///db.sqlite.denorm'
engine = create_engine(db_uri, echo=False)

# reflect sample data and create DDL
from sqlalchemy import MetaData, Table, Column
from sqlalchemy.schema import CreateTable
from migrate.changeset.constraint import PrimaryKeyConstraint, ForeignKeyConstraint
meta = MetaData()
meta.reflect(bind=engine)
# sort tables by dependency order
for t in meta.sorted_tables:
    print('-- ', t.name, ' sample data: ', engine.execute(t.select()).fetchall()) # stdout ddl
    # fact
    if t.name == fact_name:
        for c in t.columns:
            # fk
            if c.name.startswith('dim_'):
                t.append_constraint( ForeignKeyConstraint([c.name], [c.name]) )
        # rm all non relevant cols
        ddl_lines = str(CreateTable(t)).split('\n')
        for i in reversed(range(2, len(ddl_lines) - 3)):
            if not [ele for ele in ['units ','"dim_'] if(ele in ddl_lines[i])]:
                del ddl_lines[i]
        tbl_ddl = "\n".join(ddl_lines)
    # dimension
    else:
       # pk
       t.append_constraint( PrimaryKeyConstraint('_id', name=t.name + '_pk') )
       tbl_ddl = CreateTable(t)
    # todo: standardize data types to VARCHAR (except fact units)
    # todo: replace dots with underscores in column names       
    print(tbl_ddl) # stdout ddl
    # recreate table
    t.drop(engine)
    engine.execute(tbl_ddl)
