import asyncio
import grpc

from jumpstarter.v1 import rendezvous_pb2
from jumpstarter.v1 import rendezvous_pb2_grpc
from jumpstarter.v1 import jumpstarter_pb2
from jumpstarter.v1 import jumpstarter_pb2_grpc

from stream import forward


class ForClientServicer(jumpstarter_pb2_grpc.ForClientServicer):
    async def GetReport(self, request, context):
        context.set_details("dummy implementation in python")
        context.set_code(grpc.StatusCode.INTERNAL)


async def main():
    server = grpc.aio.server()

    jumpstarter_pb2_grpc.add_ForClientServicer_to_server(ForClientServicer(), server)

    # grpc.aio only supports TCP
    server.add_insecure_port("127.0.0.1:8002")

    asyncio.create_task(server.start())

    async with grpc.aio.insecure_channel("127.0.0.1:8000") as channel:
        rendezvous = rendezvous_pb2_grpc.RendezvousServiceStub(channel)
        async for resp in rendezvous.Listen(
            rendezvous_pb2.ListenRequest(address="unix:///dummy-python")
        ):
            reader, writer = await asyncio.open_connection("127.0.0.1", 8002)
            asyncio.create_task(forward(rendezvous, resp.stream, reader, writer))


asyncio.run(main())
