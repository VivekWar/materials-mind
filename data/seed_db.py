import os
import sys
import time
import pandas as pd
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
# 2. Schema Definitions
# ==========================================
TARGET_COLUMNS = [
    "category", "subcategory", "formula", "density", "melting_point", 
    "boiling_point", "thermal_conductivity", "specific_heat", "thermal_expansion", 
    "electrical_resistivity", "yield_strength", "tensile_strength", "youngs_modulus", 
    "hardness_vickers", "poissons_ratio", "glass_transition_temp", 
    "heat_deflection_temp", "processing_temp_min_c", "processing_temp_max_c", 
    "crystallinity", "fracture_toughness", "weibull_modulus", 
    "interlaminar_shear_strength", "fiber_volume_fraction", "notes"
]

UNITS = {
    "density": "g/cm3", "yield_strength": "MPa", "tensile_strength": "MPa",
    "youngs_modulus": "GPa", "melting_point": "°C", "glass_transition_temp": "°C",
    "heat_deflection_temp": "°C", "thermal_conductivity": "W/m·K"
}

# ==========================================
# 3. Helper Functions
# ==========================================
def safe_val(val):
    """Converts NaN to None so Postgres safely inserts NULL."""
    if pd.isna(val):
        return None
    return val

def build_embedding_text(row):
    """Builds a semantic string for the AI."""
    text_parts = [f"Material Name: {row.get('name', 'Unknown')}."]
    for col in TARGET_COLUMNS:
        if col in row and row[col] is not None:  
            clean_col_name = col.replace("_", " ").title()
            unit = UNITS.get(col, "")
            text_parts.append(f"{clean_col_name}: {row[col]} {unit}".strip() + ".")
    return " ".join(text_parts)

# ==========================================
# 4. Main ETL Pipeline
# ==========================================
def main():
    print("🚀 Connecting to PostgreSQL...")
    conn = psycopg2.connect(DATABASE_URL)
    
    print("📊 Loading and Merging CSV Data...")
    # Using the exact prepared file as requested
    df_main = pd.read_csv("materials_prepared_for_embedding.csv")
    
    df_metals = pd.DataFrame()
    if Path("Data.csv").exists():
        df_raw = pd.read_csv("Data.csv")
        df_metals['name'] = df_raw[['Std', 'Material', 'Heat treatment']].fillna('').agg(' '.join, axis=1)
        df_metals['category'] = 'Alloys'
        df_metals['yield_strength'] = df_raw['Sy'].astype(str).str.replace(' max', '').astype(float)
        df_metals['tensile_strength'] = df_raw['Su'].astype(float)
        df_metals['youngs_modulus'] = df_raw['E'].astype(float) / 1000 
        df_metals['density'] = df_raw['Ro'].astype(float) / 1000
        df_metals['poissons_ratio'] = df_raw['mu'].astype(float)
    
    df = pd.concat([df_main, df_metals], ignore_index=True)
    
    for col in df.columns:
        df[col] = df[col].apply(safe_val)

    records = df.to_dict(orient='records')
    print(f"⚙️ Starting ETL Pipeline for {len(records)} materials...")

    BATCH_SIZE = 100 
    
    with conn.cursor() as cursor:
        for i in tqdm(range(0, len(records), BATCH_SIZE), desc="Processing Batches"):
            batch = records[i:i + BATCH_SIZE]
            
            # --- EXTRACT & LOAD METADATA ---
            for row in batch:
                name = str(row.get('name', 'Unknown')).strip()
                cursor.execute("SELECT id FROM materials WHERE name = %s", (name,))
                result = cursor.fetchone()
                
                if result:
                    row['material_id'] = result[0]
                else:
                    cols_to_insert = ["name"]
                    vals_to_insert = [name]
                    for col in TARGET_COLUMNS:
                        if col in row and row[col] is not None:
                            cols_to_insert.append(col)
                            vals_to_insert.append(row[col])
                    
                    placeholders = ", ".join(["%s"] * len(vals_to_insert))
                    col_names = ", ".join(cols_to_insert)
                    
                    insert_sql = f"INSERT INTO materials ({col_names}) VALUES ({placeholders}) RETURNING id;"
                    cursor.execute(insert_sql, tuple(vals_to_insert))
                    row['material_id'] = cursor.fetchone()[0]

            # --- FILTER OUT EXISTING EMBEDDINGS ---
            batch_ids = tuple([row['material_id'] for row in batch])
            cursor.execute("SELECT material_id FROM material_embeddings WHERE material_id IN %s", (batch_ids,))
            existing_ids = {row[0] for row in cursor.fetchall()}
            
            records_to_embed = [r for r in batch if r['material_id'] not in existing_ids]
            
            if not records_to_embed:
                # If all records in this batch already have embeddings, skip to the next batch immediately
                conn.commit()
                continue 

            # --- GENERATE EMBEDDINGS (Batched via API) ---
            texts_to_embed = [build_embedding_text(row) for row in records_to_embed]
            
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

            # --- LOAD EMBEDDINGS INTO DB (Bulk Insert) ---
            insert_data = [
                (records_to_embed[idx]['material_id'], str(emb)) 
                for idx, emb in enumerate(batch_embeddings)
            ]
            
            execute_values(
                cursor,
                "INSERT INTO material_embeddings (material_id, embedding) VALUES %s",
                insert_data
            )

            conn.commit()
            
            # --- THE MAGIC FIX: WAIT FOR QUOTA RESET ---
            print(f"\n✅ Inserted {len(records_to_embed)} vectors. Sleeping for 60 seconds to reset API minute limits...")
            time.sleep(60)

    conn.close()
    print("\n✅ ETL Pipeline Complete! Database is fully seeded.")

if __name__ == "__main__":
    main()