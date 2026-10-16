# {{PROJECT_NAME}}

A small Python HTTP service by {{AUTHOR}}. Standard library only.

```console
python -m venv .venv && . .venv/bin/activate     # Windows: .venv\Scripts\activate
pip install -e .
{{PROJECT_NAME_KEBAB}}                           # listens on :8080 (set PORT to change it)
curl localhost:8080/healthz
python -m unittest discover -s tests
```

Needs Python 3.9 or newer.
