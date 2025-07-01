"""
Authentication Demo - Auth Gateway Features

This demo showcases the authentication and authorization features:
- JWT token generation and validation
- Role-based access control
- Token refresh
- Protected endpoints
"""
import asyncio
import aiohttp
import json
import time
import logging

logging.basicConfig(level=logging.INFO)
logger = logging.getLogger(__name__)

AUTH_URL = "http://localhost:8080"

class AuthDemo:
    """Demo for authentication features"""
    
    def __init__(self):
        self.session = None
        
    async def __aenter__(self):
        self.session = aiohttp.ClientSession()
        return self
        
    async def __aexit__(self, exc_type, exc_val, exc_tb):
        if self.session:
            await self.session.close()
    
    async def demo_login(self, username, password):
        """Demonstrate login process"""
        logger.info(f"🔐 Attempting login for user: {username}")
        
        login_data = {"username": username, "password": password}
        
        try:
            async with self.session.post(
                f"{AUTH_URL}/auth/login",
                json=login_data,
                headers={"Content-Type": "application/json"}
            ) as response:
                if response.status == 200:
                    result = await response.json()
                    logger.info(f"✅ Login successful!")
                    logger.info(f"   Token Type: {result['token_type']}")
                    logger.info(f"   Expires In: {result['expires_in']} seconds")
                    logger.info(f"   User Roles: {result['user']['roles']}")
                    logger.info(f"   Tenant ID: {result['user']['tenant_id']}")
                    return result
                else:
                    error_data = await response.json()
                    logger.error(f"❌ Login failed: {error_data.get('detail', 'Unknown error')}")
                    return None
        except Exception as e:
            logger.error(f"❌ Login error: {e}")
            return None
    
    async def demo_protected_endpoint(self, token):
        """Demonstrate accessing protected endpoint"""
        logger.info("🛡️ Accessing protected endpoint...")
        
        headers = {"Authorization": f"Bearer {token}"}
        
        try:
            async with self.session.get(f"{AUTH_URL}/auth/me", headers=headers) as response:
                if response.status == 200:
                    user_info = await response.json()
                    logger.info("✅ Protected endpoint accessed successfully!")
                    logger.info(f"   Username: {user_info['username']}")
                    logger.info(f"   Roles: {user_info['roles']}")
                    logger.info(f"   Tenant: {user_info['tenant_id']}")
                    return user_info
                else:
                    error_data = await response.json()
                    logger.error(f"❌ Protected endpoint failed: {error_data.get('detail', 'Unknown error')}")
                    return None
        except Exception as e:
            logger.error(f"❌ Protected endpoint error: {e}")
            return None
    
    async def demo_invalid_token(self):
        """Demonstrate invalid token handling"""
        logger.info("🚫 Testing invalid token...")
        
        headers = {"Authorization": "Bearer invalid_token_here"}
        
        try:
            async with self.session.get(f"{AUTH_URL}/auth/me", headers=headers) as response:
                if response.status == 401:
                    logger.info("✅ Invalid token correctly rejected!")
                    error_data = await response.json()
                    logger.info(f"   Error: {error_data.get('detail', 'Unauthorized')}")
                else:
                    logger.warning(f"⚠️ Unexpected response: {response.status}")
        except Exception as e:
            logger.error(f"❌ Invalid token test error: {e}")
    
    async def demo_no_token(self):
        """Demonstrate missing token handling"""
        logger.info("🚫 Testing missing token...")
        
        try:
            async with self.session.get(f"{AUTH_URL}/auth/me") as response:
                if response.status == 403:
                    logger.info("✅ Missing token correctly rejected!")
                else:
                    logger.warning(f"⚠️ Unexpected response: {response.status}")
        except Exception as e:
            logger.error(f"❌ Missing token test error: {e}")
    
    async def demo_health_check(self):
        """Check if auth service is healthy"""
        logger.info("🏥 Checking Auth Gateway health...")
        
        try:
            async with self.session.get(f"{AUTH_URL}/health") as response:
                if response.status == 200:
                    health_data = await response.json()
                    logger.info("✅ Auth Gateway is healthy!")
                    logger.info(f"   Status: {health_data.get('status', 'unknown')}")
                    logger.info(f"   Service: {health_data.get('service', 'unknown')}")
                    return True
                else:
                    logger.error(f"❌ Auth Gateway unhealthy: {response.status}")
                    return False
        except Exception as e:
            logger.error(f"❌ Health check failed: {e}")
            return False


async def run_auth_demo():
    """Run the complete authentication demo"""
    logger.info("🔐 Starting Authentication Demo")
    logger.info("=" * 50)
    
    async with AuthDemo() as demo:
        # 1. Health check
        is_healthy = await demo.demo_health_check()
        if not is_healthy:
            logger.error("❌ Auth Gateway is not healthy. Start it with: cd auth-gateway && python main.py")
            return
        
        print()
        
        # 2. Valid login (admin)
        admin_result = await demo.demo_login("admin", "admin123")
        if admin_result:
            admin_token = admin_result["access_token"]
            
            print()
            
            # 3. Access protected endpoint with valid token
            await demo.demo_protected_endpoint(admin_token)
        
        print()
        
        # 4. Valid login (tenant user)
        tenant_result = await demo.demo_login("tenant1", "tenant123")
        if tenant_result:
            tenant_token = tenant_result["access_token"]
            
            print()
            
            # 5. Access protected endpoint with tenant token
            await demo.demo_protected_endpoint(tenant_token)
        
        print()
        
        # 6. Invalid login
        await demo.demo_login("invalid_user", "wrong_password")
        
        print()
        
        # 7. Invalid token
        await demo.demo_invalid_token()
        
        print()
        
        # 8. Missing token
        await demo.demo_no_token()
        
        print()
        logger.info("=" * 50)
        logger.info("🎉 Authentication Demo Complete!")
        
        if admin_result:
            print()
            logger.info("💡 Sample API calls you can try:")
            token = admin_result["access_token"]
            print(f'   curl -H "Authorization: Bearer {token}" http://localhost:8080/auth/me')
            print(f'   curl -H "Authorization: Bearer {token}" http://localhost:8000/health')


if __name__ == "__main__":
    print("🔐 Authentication & Authorization Demo")
    print("=" * 45)
    print("This demo requires the Auth Gateway to be running.")
    print("Start it with: cd auth-gateway && python main.py")
    print("=" * 45)
    print()
    
    try:
        asyncio.run(run_auth_demo())
    except KeyboardInterrupt:
        print("\n👋 Demo stopped by user")
    except Exception as e:
        print(f"❌ Demo failed: {e}")
        import traceback
        traceback.print_exc()
