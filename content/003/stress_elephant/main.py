from flask import Flask, request, jsonify
import os
import psycopg2
import random
import string
import logging
import datetime
import sys

app = Flask(__name__)

# Configure logging
logging.basicConfig(level=logging.INFO, format='%(asctime)s - %(message)s', datefmt='%Y-%m-%d %H:%M:%S', handlers=[logging.StreamHandler(sys.stdout)])
logging.info("Starting API...")

@app.route('/play', methods=['POST'])
def play():
    # Use environment variables for database connection
    conn_string = f"postgresql://{os.getenv('POSTGRES_USER')}@{os.getenv('POSTGRES_HOST')}:{os.getenv('POSTGRES_PORT')}/{os.getenv('POSTGRES_DB')}?sslmode=disable"
    conn = psycopg2.connect(conn_string)
    cursor = conn.cursor()

    create_table_query = """
    CREATE TABLE IF NOT EXISTS fictitious_table (
        id SERIAL PRIMARY KEY,
        field1 TEXT,
        field2 INTEGER,
        field3 FLOAT,
        field4 DATE,
        field5 BOOLEAN
    );
    """
    cursor.execute(create_table_query)
    conn.commit()

    for _ in range(10000):
        field1 = ''.join(random.choices(string.ascii_uppercase + string.digits, k=10))
        field2 = random.randint(1, 1000)
        field3 = random.uniform(1.0, 100.0)
        field4 = f"2022-03-{random.randint(1, 31)}"
        field5 = random.choice([True, False])

        insert_query = f"""
        INSERT INTO fictitious_table (field1, field2, field3, field4, field5)
        VALUES ('{field1}', {field2}, {field3}, '{field4}', {field5});
        """
        cursor.execute(insert_query)
        conn.commit()

        logging.info(f"Inserted record: ({field1}, {field2}, {field3}, {field4}, {field5})")

    conn.close()

    return jsonify({"message": "Data inserted successfully"})

if __name__ == '__main__':
    app.run(debug=True, host='0.0.0.0', port=8888)
