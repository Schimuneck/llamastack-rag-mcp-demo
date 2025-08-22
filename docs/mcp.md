## Getting started with MCP

> this tutorial runs on MAC

1. Clone the repo:
```git clone https://github.com/Schimuneck/llamastack-rag-mcp-demo```

2. Create local venv and activate it:
```cd llamastack-rag-mcp-demo```

```uv venv .venv --python 3.12```

```source .venv/bin/activate```

3. Install requirements:
```uv pip install -r requirements.txt```

4. Start your local Postgres server:
```brew services start postgresql```

5. Allow permission to execute scripts to create PostgreSQL database and insert data:
```chmod +x scripts/create_db_and_insert_data.sh```

6. Run script to create PostgreSQL db and insert data
```./scripts/create_db_and_insert_data.sh```

7. Get data
```psql -d currys_salesforce -c "SELECT * FROM customers;"```
