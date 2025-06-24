# api/routes/internal_audit_log.py

from fastapi import APIRouter, HTTPException
from pydantic import BaseModel
from engine.writer import log_audit_event

router = APIRouter()


class AuditLogIn(BaseModel):
    action: str
    performed_by: str
    target_user: str
    details: str


@router.post("/internal/audit-log", status_code=201)
def receive_audit_log(payload: AuditLogIn):
    try:
        log_audit_event(
            action=payload.action,
            performed_by=payload.performed_by,
            target_user=payload.target_user,
            details=payload.details
        )
        return {"message": "Audit event recorded"}
    except Exception as e:
        raise HTTPException(status_code=500, detail=str(e))
