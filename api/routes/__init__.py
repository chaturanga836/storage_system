# api/routes/__init__.py

from fastapi import APIRouter
from .operations import router as operation_router
from .internal_audit_log import router as audit_log_router

router = APIRouter()

router.include_router(operation_router)
router.include_router(audit_log_router)
