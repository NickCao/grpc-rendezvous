import asyncio
import grpc

from jumpstarter.v1 import rendezvous_pb2
from jumpstarter.v1 import rendezvous_pb2_grpc
from jumpstarter.v1 import jumpstarter_pb2
from jumpstarter.v1 import jumpstarter_pb2_grpc
from google.protobuf import empty_pb2

from stream import forward


async def main():
    async def proxy():
        async with grpc.aio.insecure_channel("127.0.0.1:8000") as channel:
            rendezvous = rendezvous_pb2_grpc.RendezvousServiceStub(channel)

            async def handle(reader, writer):
                resp = await rendezvous.Dial(
                    rendezvous_pb2.DialRequest(address="unix:///dummy-python")
                )
                asyncio.create_task(forward(rendezvous, resp.stream, reader, writer))

            server = await asyncio.start_unix_server(handle, "/tmp/rendezvous-client.sock")

            async with server:
                await server.serve_forever()

    asyncio.create_task(proxy())

    await asyncio.sleep(1)

    async with grpc.aio.insecure_channel("unix:/tmp/rendezvous-client.sock") as channel:
        jumpstarter = jumpstarter_pb2_grpc.ForClientStub(channel)

        await jumpstarter.GetReport(empty_pb2.Empty())


asyncio.run(main())
