# Stage 1: render every project's Markdown -> HTML
FROM python:3.11-slim AS render

WORKDIR /docs
COPY requirements.txt .
RUN pip install --no-cache-dir -r requirements.txt

COPY generate.py .
COPY tour-agent ./tour-agent
COPY tour-planner-ai ./tour-planner-ai
COPY tour-planner-ui ./tour-planner-ui
COPY tour-commerce ./tour-commerce
COPY tour-planner-cms ./tour-planner-cms
COPY tour-agent-docs ./tour-agent-docs

RUN mkdir /output && python generate.py

# Stage 2: serve the rendered HTML
FROM nginx:1.27-alpine

COPY docs-nginx.conf /etc/nginx/conf.d/default.conf
COPY --from=render /output /usr/share/nginx/html

EXPOSE 80
