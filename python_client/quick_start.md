1. Activate venv                                                                                                                                                  ─╯
2. uv pip install -r requirements.txt  
3. OPENAI_API_KEY="YOUR_KEY" ENABLE_OPENAI=openai llama stack run run.yaml --image-type venv
4. uv run python mcp_server.py
5. Run client.ipynb
6. Extact the full output to see response from MCP