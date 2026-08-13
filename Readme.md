Cloudflare credentials removed from this file for security.

Use environment variables instead:
- CLOUDFLARE_API_TOKEN
- CLOUDFLARE_ZONE_ID
- CLOUDFLARE_DOMAIN


https://exploreragents.com/?a=bbbb

Deploy (token provided via environment only). The live domain exploreragents.com
is served by the Pages project `exploreragents` — deploying to `tour-planner-ui`
only updates the tour-planner-ui.pages.dev preview, never the live site.
Use wrangler@3 (the host's Node 20 cannot run wrangler 4).

cd /opt/tour-planner/tour-planner-ui
npm run build
npx wrangler@3 pages deploy dist --project-name exploreragents

Optional: keep the preview project in sync so old bookmarks behave identically:
npx wrangler@3 pages deploy dist --project-name tour-planner-ui

All API calls go same-origin through https://exploreragents.com/api/* — proxied
to API Gateway by tour-planner-ui/functions/api/[[path]].js (deployed together
with the site). Never point the UI at a raw *.execute-api.amazonaws.com URL.
