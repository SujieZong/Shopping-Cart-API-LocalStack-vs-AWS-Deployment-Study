"""
Locust load testing file for Product API
Supports both HttpUser and FastHttpUser for performance comparison
"""

import json
import random
from locust import HttpUser, FastHttpUser, task, between, events
import logging

# Configure logging
logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

# Configuration
MIN_PRODUCT_ID = 1
MAX_PRODUCT_ID = 100

# Sample product data templates
MANUFACTURERS = [
    "TechCorp", "Peripherals Inc", "Electronics Ltd", "Digital Devices Co",
    "Smart Tech", "Innovative Solutions", "Global Electronics", "Premium Products"
]

CATEGORIES = list(range(10, 20))

def generate_product_data(product_id):
    """Generate realistic product data for POST requests"""
    return {
        "product_id": product_id,
        "sku": f"SKU-{product_id:05d}-{random.randint(1000, 9999)}",
        "manufacturer": random.choice(MANUFACTURERS),
        "category_id": random.choice(CATEGORIES),
        "weight": random.randint(50, 5000),
        "some_other_id": random.randint(100, 999)
    }


class ProductAPIHttpUser(HttpUser):
    """
    Standard HttpUser - uses Python's requests library
    
    Characteristics:
    - Full HTTP/1.1 implementation
    - Connection pooling and keep-alive
    - Cookie handling, redirects, etc.
    - More overhead per request
    - Better for testing realistic user behavior
    - Slower but more feature-complete
    """
    
    wait_time = between(1, 3)
    
    @task(7)  # 70% of requests - GET operations (most common in e-commerce)
    def get_product(self):
        """GET /products/{productId} - Retrieve product details"""
        product_id = random.randint(MIN_PRODUCT_ID, MAX_PRODUCT_ID)
        with self.client.get(
            f"/products/{product_id}",
            name="/products/[id] (GET)",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    if data.get("product_id") == product_id:
                        response.success()
                    else:
                        response.failure(f"Product ID mismatch: expected {product_id}, got {data.get('product_id')}")
                except Exception as e:
                    response.failure(f"Invalid JSON response: {str(e)}")
            elif response.status_code == 404:
                # 404 is acceptable for non-existent products
                response.success()
            else:
                response.failure(f"Unexpected status code: {response.status_code}")
    
    @task(3)  # 30% of requests - POST operations (creating/updating products)
    def add_product_details(self):
        """POST /products/{productId}/details - Add/update product details"""
        product_id = random.randint(MIN_PRODUCT_ID, MAX_PRODUCT_ID)
        product_data = generate_product_data(product_id)
        
        with self.client.post(
            f"/products/{product_id}/details",
            json=product_data,
            name="/products/[id]/details (POST)",
            catch_response=True
        ) as response:
            if response.status_code == 204:
                response.success()
            elif response.status_code == 400:
                response.failure(f"Bad request: {response.text}")
            else:
                response.failure(f"Unexpected status code: {response.status_code}")
    
    @task(1)  # 10% of requests - Health check (monitoring)
    def health_check(self):
        """GET /health - Health check endpoint"""
        with self.client.get(
            "/health",
            name="/health (GET)",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    if data.get("status") == "healthy":
                        response.success()
                    else:
                        response.failure(f"Unhealthy status: {data}")
                except Exception as e:
                    response.failure(f"Invalid health check response: {str(e)}")
            else:
                response.failure(f"Health check failed with status: {response.status_code}")


class ProductAPIFastHttpUser(FastHttpUser):
    """
    FastHttpUser - uses geventhttpclient (optimized HTTP client)
    
    Characteristics:
    - Lightweight, gevent-based
    - Lower overhead per request
    - Faster request execution
    - Better for high-load testing
    - May not handle all HTTP edge cases
    - Can simulate more concurrent users with same resources
    """
    
    wait_time = between(1, 3)
    
    @task(7)  # 70% of requests - GET operations (most common in e-commerce)
    def get_product(self):
        """GET /products/{productId} - Retrieve product details"""
        product_id = random.randint(MIN_PRODUCT_ID, MAX_PRODUCT_ID)
        with self.client.get(
            f"/products/{product_id}",
            name="/products/[id] (GET)",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    if data.get("product_id") == product_id:
                        response.success()
                    else:
                        response.failure(f"Product ID mismatch: expected {product_id}, got {data.get('product_id')}")
                except Exception as e:
                    response.failure(f"Invalid JSON response: {str(e)}")
            elif response.status_code == 404:
                # 404 is acceptable for non-existent products
                response.success()
            else:
                response.failure(f"Unexpected status code: {response.status_code}")
    
    @task(3)  # 30% of requests - POST operations (creating/updating products)
    def add_product_details(self):
        """POST /products/{productId}/details - Add/update product details"""
        product_id = random.randint(MIN_PRODUCT_ID, MAX_PRODUCT_ID)
        product_data = generate_product_data(product_id)
        
        with self.client.post(
            f"/products/{product_id}/details",
            json=product_data,
            name="/products/[id]/details (POST)",
            catch_response=True
        ) as response:
            if response.status_code == 204:
                response.success()
            elif response.status_code == 400:
                response.failure(f"Bad request: {response.text}")
            else:
                response.failure(f"Unexpected status code: {response.status_code}")
    
    @task(1)  # 10% of requests - Health check (monitoring)
    def health_check(self):
        """GET /health - Health check endpoint"""
        with self.client.get(
            "/health",
            name="/health (GET)",
            catch_response=True
        ) as response:
            if response.status_code == 200:
                try:
                    data = response.json()
                    if data.get("status") == "healthy":
                        response.success()
                    else:
                        response.failure(f"Unhealthy status: {data}")
                except Exception as e:
                    response.failure(f"Invalid health check response: {str(e)}")
            else:
                response.failure(f"Health check failed with status: {response.status_code}")


# Event handlers for custom metrics and logging
@events.test_start.add_listener
def on_test_start(environment, **kwargs):
    """Called when a test is starting"""
    logger.info("=" * 80)
    logger.info("LOAD TEST STARTING")
    logger.info(f"Host: {environment.host}")
    logger.info(f"User class: {environment.user_classes}")
    logger.info("=" * 80)


@events.test_stop.add_listener
def on_test_stop(environment, **kwargs):
    """Called when a test is stopping"""
    logger.info("=" * 80)
    logger.info("LOAD TEST COMPLETED")
    logger.info("=" * 80)


@events.request.add_listener
def on_request(request_type, name, response_time, response_length, exception, **kwargs):
    """Log individual request details (optional - can be noisy)"""
    if exception:
        logger.warning(f"Request failed: {name} - {exception}")
