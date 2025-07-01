FROM python:3.11-slim

WORKDIR /app

# Copy requirements and install dependencies
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy shared protos
COPY ../shared-protos ./shared-protos

# Copy service code
COPY . .

# Expose ports
EXPOSE 8082 50052

# Run the service
CMD ["python", "main.py"]
