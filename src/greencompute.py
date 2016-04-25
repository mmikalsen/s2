import BaseHTTPServer
import sys
import getopt
import threading
import signal
import socket
import httplib
import random
import string
from time import sleep

httpdServeRequests = True
latencySkew = 5
latencyMultiplier = 5

def waitToProduce():
    tts = latencyMultiplier * random.expovariate(latencySkew)
    sleep(tts)
    return tts

class FrontendHttpHandler(BaseHTTPServer.BaseHTTPRequestHandler):

    # Returns the
    def do_GET(self):
  
        ttl = waitToProduce()
        print "waiting " + str(ttl).replace('.', ',') + " s"

        # Write header
        self.send_response(200)
        self.send_header("Content-type", "application/octet-stream")
        self.end_headers()

        # Write Body
        self.wfile.write("GREEN")
        self.wfile.close()

    def sendErrorResponse(self, code, msg):
        self.send_response(code)
        self.send_header("Content-type", "text/html")
        self.end_headers()
        self.wfile.write(msg)

class FrontendHTTPServer(BaseHTTPServer.HTTPServer):

    def server_bind(self):
        BaseHTTPServer.HTTPServer.server_bind(self)
        self.socket.settimeout(2)
        self.run = True

    def get_request(self):
        while self.run == True:
            try:
                sock, addr = self.socket.accept()
                sock.settimeout(2)
                return (sock, addr)
            except socket.timeout:
                if not self.run:
                    raise socket.error

    def stop(self):
        self.run = False

    def serve(self):
        while self.run == True:
            self.handle_request()

if __name__ == '__main__':
    httpserver_port = 8000

    try:
        optlist, args = getopt.getopt(sys.argv[1:], 'x', ['skew=', 'multiplier=', 'port='])
    except getopt.GetoptError:
        print sys.argv[0] + ' [--port portnumber(default=8000)] [--skew value] [--multiplier value]'
        sys.exit(2)

    for opt, arg in optlist:
        if opt in ("-skew", "--skew"):
            latencySkew = int(arg)
        elif opt in ("-multiplier", "--port"):
            latencyMultiplier = int(arg)
        elif opt in ("-port", "--port"):
            httpserver_port = int(arg)

    # Start the webserver which handles incomming requests
    try:
        httpd = FrontendHTTPServer(("",httpserver_port), FrontendHttpHandler)
        server_thread = threading.Thread(target = httpd.serve)
        server_thread.daemon = True
        server_thread.start()

        def handler(signum, frame):
            print "Stopping http server..."
            httpd.stop()
        signal.signal(signal.SIGINT, handler)

    except:
        print "Error: unable to http server thread"

    # Wait for server thread to exit
    server_thread.join(1000)
    httpd.stop()
