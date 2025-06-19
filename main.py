from fastapi import FastAPI
from api.routes.operations import router as operations_router

app = FastAPI()

app.include_router(operations_router)


# run.py

import uvicorn

if __name__ == "__main__":
    uvicorn.run("api.main:app", host="0.0.0.0", port=9090, reload=True)