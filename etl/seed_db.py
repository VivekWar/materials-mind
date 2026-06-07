import os
import time
import psycopg2
from google import genai
from psycopg2.extras import Json

# Initialize Google GenAI Client
client = genai.Client(api_key=os.environ.get("GEMINI_API_KEY"))

DB_URL = os.environ.get("POSTGRES_URL", "postgresql://postgres:password123@localhost:6432/materialmind")

def exponential_backoff(func, max_retries=5, base_delay=1):
    for attempt in range(max_retries):
        try:
            return func()
        except Exception as e:
            if "429" in str(e) or "503" in str(e):
                delay = base_delay * (2 ** attempt)
                print(f"Rate limited. Retrying in {delay}s...")
                time.sleep(delay)
            else:
                raise e
    raise Exception("Max retries exceeded")

def get_embedding(text):
    def _call():
        response = client.models.embed_content(
            model='text-embedding-004',
            contents=text,
            config={'task_type': 'RETRIEVAL_DOCUMENT'}
        )
        return response.embeddings[0].values
    return exponential_backoff(_call)

def ingest_material(conn, material):
    # Create a dense textual representation for embedding
    text_rep = f"{material['name']} is a {material['category']}. Density: {material['density']} g/cm3."

    print(f"Getting embedding for {material['name']}...")
    embedding = get_embedding(text_rep)

    with conn.cursor() as cur:
        # UPSERT Material Properties
        cur.execute("""
            INSERT INTO materials (name, category, density)
            VALUES (%s, %s, %s)
            ON CONFLICT (mp_material_id) DO UPDATE SET
                density = EXCLUDED.density
            RETURNING id;
        """, (material['name'], material['category'], material['density']))
        
        mat_id = cur.fetchone()[0]

        # UPSERT Embedding (safe update)
        cur.execute("""
            INSERT INTO material_embeddings (material_id, embedding)
            VALUES (%s, %s::vector)
            ON CONFLICT (material_id) DO UPDATE SET
                embedding = EXCLUDED.embedding,
                updated_at = NOW();
        """, (mat_id, embedding))
        
        conn.commit()
        print(f"Successfully ingested {material['name']}")

if __name__ == "__main__":
    conn = psycopg2.connect(DB_URL)
    
    dummy_materials = [
        {"name": "Titanium Ti-6Al-4V", "category": "Metal", "density": 4.43},
        {"name": "Carbon Fiber Composite", "category": "Composite", "density": 1.6}
    ]

    for m in dummy_materials:
        ingest_material(conn, m)
    
    conn.close()
