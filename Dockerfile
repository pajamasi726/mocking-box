FROM python:3.12-slim
WORKDIR /box
COPY pyproject.toml README.md ./
COPY mockingbox ./mockingbox
RUN pip install --no-cache-dir .
ENTRYPOINT ["mockingbox"]
CMD ["--help"]
