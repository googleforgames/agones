using System;
using System.Threading;
using System.Threading.Tasks;
using System.IO;
using Grpc.Core;
using Allocation;
using System.Net.Http;

namespace AllocatorClient
{
    class Program
    {
        static async Task Main(string[] args)
        {
            if (args.Length < 6) {
                throw new Exception("Arguments are missing. Expecting: <private key> <public key> <server CA> <external IP> <namepace> <enable multi-cluster>");
            }

            string clientKey    = File.ReadAllText(args[0]);
            string clientCert   = File.ReadAllText(args[1]);
            string serverCa     = File.ReadAllText(args[2]);
            string externalIp   = args[3];
            string namespaceArg = args[4];
            bool   multicluster = bool.Parse(args[5]);

            var creds = new SslCredentials(serverCa, new KeyCertificatePair(clientCert, clientKey));
            var channel = new Channel(externalIp + ":443", creds);
            var client = new AllocationService.AllocationServiceClient(channel);

           try {
                var response = await client.AllocateAsync(new AllocationRequest { 
                    Namespace = namespaceArg,
                    MultiClusterSetting = new Allocation.MultiClusterSetting {
                        Enabled = multicluster,
                    }
                });
                Console.WriteLine(response);
            } 
            catch(RpcException e)
            {
                Console.WriteLine($"gRPC error: {e}");
            }
        }
    }
}