# UDP connection test

# ### Imports ### #
import socket

# ### Configuration ### #
SERVER_IP = "127.0.0.1"
SERVER_PORT = 9998
BUFFER_SIZE = 1024

# ### Begin ### #

# Startup UDP interface
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)

# Send and receive messages
sendmessage = True
exit_message = ('q','quit','exit')

while sendmessage:
    MESSAGE = input('Send:')
    if MESSAGE not in exit_message:
        s.sendto(MESSAGE.encode(), (SERVER_IP,SERVER_PORT))
        data, addr = s.recvfrom(2048)
        print("Response:",data)
                
    else:
        sendmessage = False

# Close connection
s.close()
