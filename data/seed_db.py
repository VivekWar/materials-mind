import os
import sys
import time
import json
import psycopg2
from psycopg2.extras import execute_values
from google import genai
from google.genai import types
from tqdm import tqdm
from pathlib import Path

# ==========================================
# 1. Configuration & Security (.env loading)
# ==========================================
_env_file = Path(__file__).parent.parent / ".env"  
if _env_file.exists():
    for _line in _env_file.read_text().splitlines():
        _line = _line.strip()
        if _line and not _line.startswith("#") and "=" in _line:
            _k, _v = _line.split("=", 1)
            os.environ.setdefault(_k.strip(), _v.strip())

GEMINI_API_KEY = os.getenv("GEMINI_API_KEY")
DATABASE_URL = os.getenv("DATABASE_URL")

if not GEMINI_API_KEY or not DATABASE_URL:
    print("❌ ERROR: GEMINI_API_KEY and DATABASE_URL must be set in your .env file.")
    sys.exit(1)

client = genai.Client(api_key=GEMINI_API_KEY)

# ==========================================
# 2. Main ETL Pipeline
# ==========================================
def main():
    print("🚀 Connecting to PostgreSQL...")
    conn = psycopg2.connect(DATABASE_URL)
    
    print("📊 Loading JSON Data...")
    json_path = Path("material_embeddings_data.json")
    if not json_path.exists():
        # Fallback to look inside data dir if executed from root
        json_path = Path("data/material_embeddings_data.json")
        
    with open(json_path, 'r', encoding='utf-8') as f:
        records = json.load(f)

    print(f"⚙️ Starting ETL Pipeline for {len(records)} materials...")

    BATCH_SIZE = 100 
    
    with conn.cursor() as cursor:
        for i in tqdm(range(0, len(records), BATCH_SIZE), desc="Processing Batches"):
            batch = records[i:i + BATCH_SIZE]
            
            # --- FILTER OUT EXISTING RECORDS ---
            # Check which names already exist in the database
            batch_names = tuple([str(row.get('name', '')).strip() for row in batch])
            cursor.execute("SELECT name FROM materials WHERE name IN %s", (batch_names,))
            existing_names = {row[0] for row in cursor.fetchall()}
            
            records_to_process = [r for r in batch if str(r.get('name', '')).strip() not in existing_names]
            
            if not records_to_process:
                # If all records in this batch already exist, skip to the next batch immediately
                conn.commit()
                continue 

            # --- GENERATE EMBEDDINGS (Batched via API) ---
            texts_to_embed = []
            for item in records_to_process:
                embedding_text = f"Material: {item.get('name', 'Unknown')}. Category: {item.get('Category', 'Unknown')} - {item.get('subcategory', 'Unknown')}. Applications: {item.get('usage_information', 'None')}"
                texts_to_embed.append(embedding_text)
            
            try:
                # Fire all texts at once (Uses exactly your 100 RPM limit)
                response = client.models.embed_content(
                    model='gemini-embedding-001',
                    contents=texts_to_embed,
                    config=types.EmbedContentConfig(output_dimensionality=768), 
                )
                batch_embeddings = [e.values for e in response.embeddings]
                
            except Exception as api_err:
                print(f"\n⚠️ Fatal API Error: {api_err}")
                print("🛑 The script was forced to stop to protect your database. Try running again.")
                conn.close()
                sys.exit(1)

            # --- LOAD INTO DB (Bulk Insert) ---
            insert_data = []
            for idx, item in enumerate(records_to_process):
                name = str(item.get('name', 'Unknown')).strip()
                formula = item.get('formula')
                category = item.get('Category')
                subcategory = item.get('subcategory')
                notes = item.get('usage_information')
                
                all_props = item.get('all_properties', {})
                specific_properties = json.dumps(all_props)
                
                embedding_vector = str(batch_embeddings[idx])
                
                insert_data.append((
                    name, formula, category, subcategory, specific_properties, notes, embedding_vector
                ))
            
            execute_values(
                cursor,
                """
                INSERT INTO materials (
                    name, formula, category, subcategory, specific_properties, notes, embedding
                ) VALUES %s
                """,
                insert_data
            )

            conn.commit()
            
            # --- THE MAGIC FIX: WAIT FOR QUOTA RESET ---
            print(f"\n✅ Inserted {len(records_to_process)} vectors. Sleeping for 60 seconds to reset API minute limits...")
            time.sleep(60)

    conn.close()
    print("\n✅ ETL Pipeline Complete! Database is fully seeded.")

if __name__ == "__main__":
    main()