export default {
  async fetch(request, env, ctx) {
    if (request.method === "OPTIONS") {
      return new Response(null, {
        headers: {
          "Access-Control-Allow-Origin": "*",
          "Access-Control-Allow-Methods": "POST, OPTIONS",
          "Access-Control-Allow-Headers": "Content-Type",
        },
      });
    }

    if (request.method !== "POST") {
      return new Response("Method Not Allowed", { status: 405 });
    }

    let payload;
    try {
      payload = await request.json();
    } catch (e) {
      return new Response("Invalid JSON", { status: 400 });
    }

    if (!Array.isArray(payload) || payload.length === 0) {
      return new Response("Payload must be a non-empty array of strings", {
        status: 400,
      });
    }

    // 缓存逻辑：尝试从 Cache 中获取
    // 使用 payload 内容（排序后）作为唯一 Key
    const cache = caches.default;
    const sortedPayload = [...payload].sort();
    const cacheUrl = new URL(request.url);
    // 构造一个虚拟的 GET 请求作为 Cache Key
    const cacheKey = new Request(
      `https://${cacheUrl.hostname}/api/cached-v1/${sortedPayload.join(",")}`,
      {
        method: "GET",
      }
    );

    let cachedResponse = await cache.match(cacheKey);
    if (cachedResponse) {
      return cachedResponse;
    }

    const formData = new FormData();
    formData.append("itemcount", payload.length.toString());
    payload.forEach((id, index) => {
      formData.append(`publishedfileids[${index}]`, id);
    });

    try {
      // 用于存放从页面爬取的依赖 ID (仅当请求单个 ID 时尝试爬取)
      let scrapedDependencies = [];
      let fetchPagePromise = Promise.resolve(null);

      // 如果只请求了一个 ID，则尝试爬取该页面的依赖信息
      if (payload.length === 1) {
        const targetId = payload[0];
        fetchPagePromise = fetch(
          `https://steamcommunity.com/sharedfiles/filedetails/?id=${targetId}`,
          {
            headers: {
              "User-Agent":
                "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/91.0.4472.124 Safari/537.36",
            },
          }
        )
          .then((res) => {
            if (res.ok) return res.text();
            return null;
          })
          .catch(() => null);
      }

      const [steamResponse, pageHtml] = await Promise.all([
        fetch(
          "https://api.steampowered.com/ISteamRemoteStorage/GetPublishedFileDetails/v1/",
          {
            method: "POST",
            body: formData,
          }
        ),
        fetchPagePromise,
      ]);

      // 处理 HTML 获取依赖
      let debugInfo = {
        htmlLength: 0,
        containerFound: false,
        block: "",
        scraped: [],
        fetchError: null,
      };

      if (pageHtml) {
        debugInfo.htmlLength = pageHtml.length;

        // 策略：优先寻找 requiredItemsContainer，如果找不到则在看似是侧边栏的区域搜索，最后尝试全文
        const startMarker = '<div class="requiredItemsContainer">';
        let startIndex = pageHtml.indexOf(startMarker);
        let searchScope = pageHtml;

        if (startIndex !== -1) {
          debugInfo.containerFound = true;
          // 找到了容器，就在之后的内容中搜索，截取一段足够长的
          searchScope = pageHtml.substring(startIndex, startIndex + 10000);
        } else {
          // 没找到容器，尝试寻找 "Required items" 文本 (可能是英文界面)
          // 或者 "id=\"RequiredItems\"" 这种可能的 ID
          const backupStart = pageHtml.indexOf("RequiredItems");
          if (backupStart !== -1) {
            // 找到了类似的 ID 或 class 名
            searchScope = pageHtml.substring(backupStart, backupStart + 10000);
          }
          // 如果都没找到，searchScope 依然是 pageHtml (全文搜索模式)
        }

        debugInfo.block = searchScope.substring(0, 200); // 调试看开头

        // 正则：匹配 href=".../filedetails/?id=123..."
        // 这种格式比较通用，能匹配 workshop 和 sharedfiles 的链接
        const linkRegex = /href="[^"]*\/filedetails\/\?id=(\d+)/g;

        let match;
        // 为了防止在全文搜索模式下抓取到太多无关链接（如“作者的其他物品”、“最近查看”等），
        // 全文模式下可能需要更严格的过滤，但目前先抓取全部并在客户端/人工分辨
        // 通常依赖项会在页面中部或右侧

        while ((match = linkRegex.exec(searchScope)) !== null) {
          const foundId = match[1];
          // 防重 + 排除自己
          // 这里的 payload[0] 可能是数字，foundId 是 regex 出来的字符串
          // 因此 !== 判断会失效，导致把自己加进去
          if (
            String(foundId) !== String(payload[0]) &&
            !scrapedDependencies.includes(foundId)
          ) {
            scrapedDependencies.push(foundId);
          }
        }

        debugInfo.scraped = scrapedDependencies;
      } else {
        debugInfo.fetchError = "Page HTML is null (fetch failed)";
      }

      console.log(JSON.stringify(debugInfo)); // 将调试信息打印到控制台 (Cloudflare Worker Logs)

      if (!steamResponse.ok) {
        return new Response(`Steam API Error: ${steamResponse.status}`, {
          status: 502,
        });
      }

      const steamData = await steamResponse.json();

      if (
        !steamData ||
        !steamData.response ||
        !steamData.response.publishedfiledetails
      ) {
        return new Response("Invalid response from Steam", { status: 502 });
      }

      const mappedData = steamData.response.publishedfiledetails.map((item) => {
        // 获取 API 返回的 children，并精简字段
        let rawChildren = item.children || [];
        let finalChildren = rawChildren.map((c) => ({
          publishedfileid: c.publishedfileid,
        }));

        // 如果是主请求的物品（通常 result count 为 1 或者匹配 ID），且我们爬取到了依赖
        // 我们将依赖也加入到 children 中，这样客户端会一并下载它们
        // 修正：确保转换为字符串比较，因为 API 返回的 ID 是字符串，而 payload 可能是数字
        if (
          payload.length === 1 &&
          String(payload[0]) === String(item.publishedfileid) &&
          scrapedDependencies.length > 0
        ) {
          // 转换为 WorkshopChild 格式
          const existingIds = new Set(
            finalChildren.map((c) => c.publishedfileid)
          );

          scrapedDependencies.forEach((depId) => {
            if (!existingIds.has(depId)) {
              finalChildren.push({
                publishedfileid: depId,
              });
            }
          });
        }

        const resultItem = {
          result: item.result,
          publishedfileid: item.publishedfileid,
          filename: item.filename,
          file_size: item.file_size,
          file_url: item.file_url,
          preview_url: item.preview_url,
          title: item.title,
          children: finalChildren,
        };

        // 过滤合集/依赖父项本身的图片文件
        // 如果是有子项的物品（合集或有依赖），且它自己的文件名是图片格式，
        // 将其 result 设为 0，防止客户端下载这个图片
        if (finalChildren.length > 0) {
          const lowerName = (item.filename || "").toLowerCase();
          const isImage =
            lowerName.endsWith(".jpg") ||
            lowerName.endsWith(".jpeg") ||
            lowerName.endsWith(".png") ||
            lowerName.endsWith(".gif");

          if (isImage) {
            resultItem.result = 0;
          }
        }

        return resultItem;
      });

      const response = new Response(JSON.stringify(mappedData), {
        headers: {
          "Content-Type": "application/json",
          "Access-Control-Allow-Origin": "*",
          "Cache-Control": "public, max-age=3600",
        },
      });

      // 写入缓存，不阻塞主线程
      ctx.waitUntil(cache.put(cacheKey, response.clone()));

      return response;
    } catch (error) {
      return new Response(`Internal Server Error: ${error.message}`, {
        status: 500,
      });
    }
  },
};
