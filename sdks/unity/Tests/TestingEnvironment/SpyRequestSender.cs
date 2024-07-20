using System.Threading.Tasks;
using Agones;
using UnityEngine.Networking;

namespace Tests.TestingEnvironment
{
    public class SpyRequestSender: IRequestSender
    {
        public string LastApi { get; private set; }
        public string LastJson { get; private set; }
        public string LastMethod { get; private set; }
        public async Task<AgonesSdk.AsyncResult> SendRequestAsync(string api, string json,
            string method = UnityWebRequest.kHttpVerbPOST)
        {
            LastApi = api;
            LastJson = json;
            LastMethod = method;
            return new AgonesSdk.AsyncResult
            {
                ok = true,
                json = "{}"
            };
        }
    }
}
