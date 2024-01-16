# We need a background HTTP server to serve packages from
import http.server
import threading
import socketserver
from functools import partial


def handler_for_dir(dir, *args, **kwargs):
    return partial(http.server.SimpleHTTPRequestHandler, directory=dir, *args, **kwargs)

class PackageServer:
    def __init__(self, dir):
        self.handler = handler_for_dir(dir)

    def __enter__(self):
        self.server = socketserver.TCPServer(("", 0), self.handler)
        self.port = self.server.server_address[1]
        self.server_thread = threading.Thread(target=self.server.serve_forever)
        self.server_thread.start()
        return self

    def __exit__(self, exc_type, exc_value, traceback):
        self.server.shutdown()
        self.server_thread.join()

    def url(self, package):
        return f'http://localhost:{self.port}/{package}'
