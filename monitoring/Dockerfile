FROM python:3.11-slim

WORKDIR /app

# Copy requirements and install dependencies
COPY monitoring/requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

# Copy service code
COPY monitoring/ .

# Expose ports
EXPOSE 8084

# Run the service
CMD ["python", "main.py"]
