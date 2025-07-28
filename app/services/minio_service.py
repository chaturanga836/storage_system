# app/services/minio_service.py

import boto3
from botocore.exceptions import ClientError
import logging
import os

logger = logging.getLogger(__name__)

class MinioService:
    def __init__(self, endpoint_url: str, access_key: str, secret_key: str, region_name: str = "us-east-1"):
        """
        Initializes the MinioService with S3 client.
        :param endpoint_url: The URL of the MinIO server (e.g., "http://localhost:9000").
        :param access_key: MinIO access key.
        :param secret_key: MinIO secret key.
        :param region_name: S3 region name (can be a placeholder like "us-east-1").
        """
        self.endpoint_url = endpoint_url
        self.access_key = access_key
        self.secret_key = secret_key
        self.region_name = region_name
        self._s3_client = None

    def get_s3_client(self):
        """
        Returns an initialized boto3 S3 client for MinIO.
        Caches the client instance.
        """
        if self._s3_client is None:
            try:
                self._s3_client = boto3.client(
                    's3',
                    endpoint_url=self.endpoint_url,
                    aws_access_key_id=self.access_key,
                    aws_secret_access_key=self.secret_key,
                    region_name=self.region_name,
                    config=boto3.session.Config(signature_version='s3v4') # Important for MinIO
                )
                logger.info("MinIO S3 client initialized.")
            except Exception as e:
                logger.error(f"Failed to initialize MinIO S3 client: {e}")
                raise
        return self._s3_client

    def create_bucket_if_not_exists(self, bucket_name: str):
        """
        Creates an S3 bucket if it does not already exist.
        :param bucket_name: The name of the bucket to create.
        """
        s3_client = self.get_s3_client()
        try:
            s3_client.head_bucket(Bucket=bucket_name)
            logger.info(f"Bucket '{bucket_name}' already exists.")
        except ClientError as e:
            error_code = int(e.response['Error']['Code'])
            if error_code == 404:
                try:
                    s3_client.create_bucket(Bucket=bucket_name)
                    logger.info(f"Bucket '{bucket_name}' created successfully.")
                except ClientError as ce:
                    logger.error(f"Failed to create bucket '{bucket_name}': {ce}")
                    raise
            else:
                logger.error(f"Error checking bucket '{bucket_name}': {e}")
                raise
        except Exception as e:
            logger.error(f"Unexpected error with bucket '{bucket_name}': {e}")
            raise


    def upload_file(self, file_path: str, bucket_name: str, object_name: str):
        """
        Uploads a file to an S3 bucket.
        :param file_path: The path to the file to upload.
        :param bucket_name: The name of the target bucket.
        :param object_name: The desired object key in the bucket.
        """
        s3_client = self.get_s3_client()
        try:
            s3_client.upload_file(file_path, bucket_name, object_name)
            logger.info(f"File '{file_path}' uploaded to s3://{bucket_name}/{object_name}")
        except ClientError as e:
            logger.error(f"Failed to upload file '{file_path}' to s3://{bucket_name}/{object_name}: {e}")
            raise
        except Exception as e:
            logger.error(f"Unexpected error uploading file: {e}")
            raise

    def download_file(self, bucket_name: str, object_name: str, download_path: str):
        """
        Downloads a file from an S3 bucket.
        :param bucket_name: The name of the source bucket.
        :param object_name: The object key to download.
        :param download_path: The local path to save the downloaded file.
        """
        s3_client = self.get_s3_client()
        try:
            s3_client.download_file(bucket_name, object_name, download_path)
            logger.info(f"Object s3://{bucket_name}/{object_name} downloaded to '{download_path}'")
        except ClientError as e:
            logger.error(f"Failed to download object s3://{bucket_name}/{object_name} to '{download_path}': {e}")
            raise
        except Exception as e:
            logger.error(f"Unexpected error downloading file: {e}")
            raise

# Example usage (would typically be in main.py or another service)
if __name__ == "__main__":
    logging.basicConfig(level=logging.INFO)

    # These would typically come from config.py
    minio_endpoint = "http://localhost:9000"
    minio_access_key = "minioadmin"
    minio_secret_key = "minioadmin"
    iceberg_warehouse_bucket = "iceberg-warehouse" # This is defined in pyiceberg.yaml

    minio_service = MinioService(minio_endpoint, minio_access_key, minio_secret_key)

    try:
        print(f"\n--- Ensuring MinIO bucket '{iceberg_warehouse_bucket}' exists ---")
        minio_service.create_bucket_if_not_exists(iceberg_warehouse_bucket)

        # Example of uploading a dummy file
        dummy_file_path = "dummy_data.txt"
        with open(dummy_file_path, "w") as f:
            f.write("This is a test file for MinIO service.\n")
        
        print(f"\n--- Uploading '{dummy_file_path}' to MinIO ---")
        minio_service.upload_file(dummy_file_path, iceberg_warehouse_bucket, "test_files/dummy_upload.txt")

        # Example of downloading the dummy file
        downloaded_file_path = "downloaded_dummy_data.txt"
        print(f"\n--- Downloading 'test_files/dummy_upload.txt' from MinIO ---")
        minio_service.download_file(iceberg_warehouse_bucket, "test_files/dummy_upload.txt", downloaded_file_path)

        # Clean up dummy files
        os.remove(dummy_file_path)
        os.remove(downloaded_file_path)
        print("\nCleaned up dummy files.")

    except Exception as e:
        print(f"MinioService test failed: {e}")