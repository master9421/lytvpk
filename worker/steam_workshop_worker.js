// Cloudflare Worker Logic for Steam Workshop Proxy
// 部署前请确保在 Cloudflare 后台配置环境变量: STEAM_API_KEY

export default {
  async fetch(request, env, ctx) {
    const url = new URL(request.url);
    const path = url.pathname;

    // 处理 CORS (允许跨域)
    if (request.method === "OPTIONS") {
      return new Response(null, {
        headers: {
          "Access-Control-Allow-Origin": "*",
          "Access-Control-Allow-Methods": "GET, POST, OPTIONS",
          "Access-Control-Allow-Headers": "Content-Type",
        },
      });
    }

    const corsHeaders = {
      "Access-Control-Allow-Origin": "*",
      "Content-Type": "application/json",
    };

    try {
      // 1. 列表 & 筛选 & 搜索
      // GET /list?q=xxx&page=1&sort=trend&tags=Weapon,Map
      if (path === "/list") {
        return await handleList(url, env, corsHeaders);
      }

      // 2. 详情
      // GET /detail?id=123456
      if (path === "/detail") {
        return await handleDetail(url, env, corsHeaders);
      }

      // 404
      return new Response(JSON.stringify({ error: "Not Found" }), {
        status: 404,
        headers: corsHeaders,
      });
    } catch (err) {
      return new Response(JSON.stringify({ error: err.message }), {
        status: 500,
        headers: corsHeaders,
      });
    }
  },
};

// ----------------------------------------------------
// Handler 1: 列表查询 (IPublishedFileService/QueryFiles)
// ----------------------------------------------------
async function handleList(url, env, headers) {
  const steamApiUrl = new URL(
    "https://api.steampowered.com/IPublishedFileService/QueryFiles/v1/"
  );

  const params = steamApiUrl.searchParams;
  params.set("key", env.STEAM_API_KEY);
  params.set("appid", "550"); // L4D2
  params.set("return_details", "true"); // 返回详情
  params.set("numperpage", "20");
  params.set("cache_max_age_seconds", "300"); // 其实 Steam 忽略这个，但我们可以加

  // 转发参数
  const q = url.searchParams.get("q");
  if (q) {
    params.set("search_text", q);
    params.set("query_type", "12"); // 12-RankedByTextSearch 如果有搜索词
  } else {
    const sort = url.searchParams.get("sort") || "trend";
    // 映射排序
    // 0: RankedByVote 1: RankedByPublicationDate 2: RankedByAcceptedForGame
    // 3: RankedByTrend 4: RankedByTotalUniqueSubscriptions 5: RankedByAccountID
    // ... 查看 Steam 文档
    // 常用: trend -> 3, recent -> 1, top -> 4?
    // QueryFiles sort order:
    // 1 = trend most recently updated? No.
    // Let's use the standard enumeration for QueryFiles:
    // k_PublishedFileQueryType_RankedByVote = 0
    // k_PublishedFileQueryType_RankedByPublicationDate = 1
    // k_PublishedFileQueryType_RankedByAcceptedForGame = 2
    // k_PublishedFileQueryType_RankedByTrend = 3
    // k_PublishedFileQueryType_RankedByTotalUniqueSubscriptions = 4

    // 注意：Steam API "query_type" 的定义有点混乱，QueryFiles v1 经常混用。
    // 为了保险，我们用最常用的：
    if (sort === "recent") params.set("query_type", "1"); // Date
    else if (sort === "top") params.set("query_type", "0"); // Vote
    else params.set("query_type", "3"); // Trend (Default)
  }

  const page = url.searchParams.get("page") || "0";
  params.set("page", page);

  // Tags 过滤
  // 格式: tags=Weapon,Map (逗号分隔)
  const tagsStr = url.searchParams.get("tags");
  if (tagsStr) {
    const tags = tagsStr.split(",");
    tags.forEach((tag, index) => {
      params.set(`requiredtags[${index}]`, tag);
    });
    params.set("match_all_tags", "true");
  }

  const resp = await fetch(steamApiUrl.toString());
  const data = await resp.json();

  return new Response(JSON.stringify(data), { headers });
}

// ----------------------------------------------------
// Handler 2: 单个详情 (ISteamRemoteStorage/GetPublishedFileDetails)
// ----------------------------------------------------
async function handleDetail(url, env, headers) {
  const id = url.searchParams.get("id");
  if (!id) throw new Error("Missing id parameter");

  const steamApiUrl =
    "https://api.steampowered.com/ISteamRemoteStorage/GetPublishedFileDetails/v1/";

  // 该接口通过 POST form-data 传递参数
  const formData = new FormData();
  formData.append("key", env.STEAM_API_KEY);
  formData.append("itemcount", "1");
  formData.append("publishedfileids[0]", id);

  const resp = await fetch(steamApiUrl, {
    method: "POST",
    body: formData,
  });

  const data = await resp.json();
  return new Response(JSON.stringify(data), { headers });
}
