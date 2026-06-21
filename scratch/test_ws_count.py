import asyncio
import websockets
import json

async def run_test():
    uri = "ws://localhost:8080/ws/testroom"
    
    # 1. Connect Host
    async with websockets.connect(uri) as ws_host:
        print("Host connected to WS")
        
        # Send JOIN_EVENT for Host
        join_host = {
            "action": "JOIN_EVENT",
            "payload": {
                "roomId": "testroom",
                "username": "HostUser",
                "isHost": True
            }
        }
        await ws_host.send(json.dumps(join_host))
        
        # Receive ROOM_INIT
        resp_host_init = await ws_host.recv()
        print("Host received:", json.loads(resp_host_init))
        
        # 2. Connect Guest
        async with websockets.connect(uri) as ws_guest:
            print("\nGuest connected to WS")
            
            # Send JOIN_EVENT for Guest
            join_guest = {
                "action": "JOIN_EVENT",
                "payload": {
                    "roomId": "testroom",
                    "username": "GuestUser",
                    "isHost": False
                }
            }
            await ws_guest.send(json.dumps(join_guest))
            
            # Host should receive JOIN_EVENT for Guest
            resp_host_join = await ws_host.recv()
            print("Host received:", json.loads(resp_host_join))
            
            # Guest should receive ROOM_INIT
            resp_guest_init = await ws_guest.recv()
            print("Guest received:", json.loads(resp_guest_init))
            
            # Check if Guest receives anything else
            try:
                # Wait 1s to see if Guest receives the broadcast JOIN_EVENT as well
                resp_guest_extra = await asyncio.wait_for(ws_guest.recv(), timeout=1.0)
                print("Guest received extra:", json.loads(resp_guest_extra))
            except asyncio.TimeoutError:
                print("Guest received nothing else (as expected).")

if __name__ == "__main__":
    # Start the event loop
    asyncio.run(run_test())
