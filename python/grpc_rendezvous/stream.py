from jumpstarter.v1 import rendezvous_pb2


async def forward(rendezvous, stream, reader, writer):
    async def rx():
        while True:  # FIXME: exit condition
            payload = await reader.read(1024)
            yield rendezvous_pb2.Frame(payload=payload)

    async for frame in rendezvous.Stream(rx(), metadata=(("stream", stream),)):
        writer.write(frame.payload)
        await writer.drain()

    writer.close()
    await writer.wait_closed()
