using System.Threading.Tasks;
using UnityEngine.Networking;

namespace Agones
{
    public interface IRequestSender
    {
        Task<AgonesSdk.AsyncResult> SendRequestAsync(string api, string json, string method = UnityWebRequest.kHttpVerbPOST);
    }
}
