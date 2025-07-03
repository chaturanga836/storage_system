FROM python:3.11-slim

WORKDIR /app

# Copy requirements and install dependencies
COPY metadata-catalog/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy shared protos
COPY shared-protos ./shared-protos

# Copy service code
COPY metadata-catalog/ .

# Expose ports
EXPOSE 8083 50053

# Run the service
CMD ["python", "main.py"]
