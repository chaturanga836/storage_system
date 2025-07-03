FROM python:3.11-slim

WORKDIR /app

# Copy requirements and install dependencies
COPY operation-node/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy shared protos
COPY shared-protos ./shared-protos

# Copy service code
COPY operation-node/ .

# Expose ports
EXPOSE 8081 50054

# Run the service
CMD ["python", "main.py"]
