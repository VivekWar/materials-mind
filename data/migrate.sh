#!/bin/bash

# Load env variables
export $(grep -v '^#' .env | xargs)

TABLES=("users" "materials" "chats" "messages" "query_log" "material_embeddings")

for TABLE in "${TABLES[@]}"; do
    echo "Exporting table $TABLE from local DB..."
    docker exec materialmind_postgres pg_dump -U postgres -d materialmind -a --column-inserts -t $TABLE -f /tmp/${TABLE}_dump.sql
    docker cp materialmind_postgres:/tmp/${TABLE}_dump.sql data/${TABLE}_dump.sql
    
    echo "Importing table $TABLE to Neon DB..."
    psql $DATABASE_URL -f data/${TABLE}_dump.sql
done

echo "Migration Complete!"
