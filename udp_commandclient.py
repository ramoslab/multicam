# TCP connection test

# ### Imports ### #
import socket

# ### Configuration ### #
SERVER_IP = "127.0.0.1"
SERVER_PORT = 9999
BUFFER_SIZE = 1024

# ### Begin ### #

# Startup TCP interface
s = socket.socket(socket.AF_INET, socket.SOCK_DGRAM)
#s.connect((SERVER_IP, SERVER_PORT))

# Send and receive messages
sendmessage = True
exit_message = ('q','quit','exit')

while sendmessage:
	MESSAGE = input('Send:')
	if MESSAGE not in exit_message:
		s.sendto(MESSAGE.encode(), (SERVER_IP,SERVER_PORT))
	else:
		sendmessage = False

	# Do not expect feedback message from server if the DATA command was sent
	#if MESSAGE.split(" ")[0] != 'DATA':
		#data = s.recv(BUFFER_SIZE)
		#print("Received:", data.decode().split(":")[1])

# Close connection
s.close()
