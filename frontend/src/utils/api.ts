export const GLOBAL_HEADERS: { [key: string]: string | undefined } = {};
const RESPONSE_HANDLERS: { [key: string]: Function[] } = { all: [] };
const REQUEST_HANDLERS = [] as Array<Function>;

type Method = "DELETE" | "GET" | "HEAD" | "OPTIONS" | "PATCH" | "POST" | "PUT";
type UrlParams = {
  [key: string]:
    | string
    | number
    | boolean
    | null
    | Array<string | number | boolean>;
} | null;

interface HandlerParamsBase {
  url: string;
  method: Method;
  json: any;
  params: UrlParams;
  headers: { [key: string]: string | undefined };
  options: { [key: string]: any };
}

export interface RequestHandlerParams extends HandlerParamsBase {
  body: BodyInit | null;
}

export interface ResponseHandlerParams extends HandlerParamsBase {
  response: Response;
  body: any;
}

let apiRoot = "";
if (import.meta.env.VITE_APP_API_URL) {
  apiRoot = import.meta.env.VITE_APP_API_URL || "";
}
const ORIG_API_ROOT = apiRoot;

function objectToQueryString(obj: {
  [key: string]:
    | string
    | number
    | boolean
    | null
    | Array<string | number | boolean>;
}): string {
  return Object.keys(obj)
    .map((key) => `${key}=${obj[key]}`)
    .join("&");
}

export function setURLRoot(root: string): void {
  apiRoot = root;
}

export function setGlobalRequestHeader(
  header: string,
  value: string | undefined = undefined,
): void {
  GLOBAL_HEADERS[header] = value;
}

export function addResponseHandler(
  status: number | "all",
  handler: Function,
): void {
  if (!RESPONSE_HANDLERS[status]) {
    RESPONSE_HANDLERS[status] = [];
  }
  RESPONSE_HANDLERS[status].push(handler);
}

export function removeResponseHandler(
  status: number | "all",
  handler: Function,
): void {
  if (RESPONSE_HANDLERS[status]) {
    const idx = RESPONSE_HANDLERS[status].indexOf(handler);
    if (idx > -1) {
      RESPONSE_HANDLERS[status].splice(idx, 1);
    }
  }
}

export function addRequestHandler(handler: Function): void {
  REQUEST_HANDLERS.push(handler);
}

export function removeRequestHandler(handler: Function): void {
  const idx = REQUEST_HANDLERS.indexOf(handler);
  REQUEST_HANDLERS.splice(idx, 1);
}

export function resetModifications(): void {
  // console.log(apiRoot, GLOBAL_HEADERS, RESPONSE_HANDLERS, REQUEST_HANDLERS);
  apiRoot = ORIG_API_ROOT;
  Object.keys(GLOBAL_HEADERS).forEach((k) => delete GLOBAL_HEADERS[k]);
  Object.keys(RESPONSE_HANDLERS).forEach((k) => delete RESPONSE_HANDLERS[k]);
  RESPONSE_HANDLERS.all = [];
  REQUEST_HANDLERS.length = 0;
}

export interface ApiResponse {
  ok: boolean;
  status: number;
  body?: any;
  error?: any;
}

function getFilenameFromContentDispositionHeader(header: string): string {
  const utf8FilenameRegex = /filename\*=UTF-8''([\w%\-.]+)(?:; ?|$)/i;
  const asciiFilenameRegex = /^filename=(["']?)(.*?[^\\])\1(?:; ?|$)/i;

  let filename = "";

  if (utf8FilenameRegex.test(header)) {
    const utf8FilenameResult = utf8FilenameRegex.exec(header);

    if (utf8FilenameResult) {
      filename = decodeURIComponent(utf8FilenameResult[1] ?? '');
    }
  } else {
    const filenameStart = header.toLowerCase().indexOf("filename=");

    if (filenameStart >= 0) {
      const partialHeader = header.slice(filenameStart);
      const matches = asciiFilenameRegex.exec(partialHeader);

      if (matches != null && matches[2]) {
        filename = matches[2];
      }
    }
  }

  return filename;
}

export default async function api(
  {
    url,
    method = "GET",
    body = null,
    json = null,
    params = null,
    headers = {},
    options = {},
  }: {
    url: string;
    method?: Method;
    body?: BodyInit | null;
    json?: any;
    params?: UrlParams;
    headers?: { [key: string]: string | undefined };
    options?: { [key: string]: any };
  } = { url: "." },
): Promise<ApiResponse> {
  REQUEST_HANDLERS.forEach((handler) => {
    handler({
      url,
      method,
      body,
      json,
      params,
      headers,
      options,
    } as RequestHandlerParams);
  });
  const init: { [key: string]: any } = {
    method,
    headers,
  };

  // add global headers if not assigned to something else
  // eslint-disable-next-line array-callback-return
  Object.keys(GLOBAL_HEADERS)
    .filter((key) => !headers[key])
    .map((key) => {
      init.headers[key] = GLOBAL_HEADERS[key];
    });

  // default to application json if content type not set and using json
  if (json && !init.headers["Content-Type"] && method !== "GET") {
    init.headers["Content-Type"] = "application/json";
  }

  if (body !== null) {
    init.body = body;
  } else if (json !== null) {
    init.body = JSON.stringify(json);
  }

  // eslint-disable-next-line array-callback-return
  Object.keys(options)
    .filter((key) => !init[key])
    .map((key) => {
      init[key] = options[key];
    });

  let urlToFetch;

  if (!url.startsWith("http:") && !url.startsWith("https:")) {
    urlToFetch = `${apiRoot}${url}`;
  } else {
    urlToFetch = url;
  }

  if (params) {
    urlToFetch += `?${objectToQueryString(params)}`;
  }

  let response: Response;

  try {
    response = await fetch(urlToFetch, init);
  } catch (e) {
    return {
      error: e,
      status: 0,
      ok: false,
    };
  }

  let responseBody: any;

  const contentType = response.headers.get("content-type");
  const contentDisposition = response.headers.get("content-disposition");

  if (contentDisposition && contentDisposition.startsWith("attachment")) {
    try {
      responseBody = await response.blob();
    } catch (e) {}

    if (responseBody) {
      responseBody = {
        filename: getFilenameFromContentDispositionHeader(contentDisposition),
        type: contentType || "",
        blob: responseBody,
      };
    }
  } else {
    try {
      responseBody = await response.text();
    } catch (e) {}

    if (
      contentType &&
      contentType.indexOf("application/json") !== -1 &&
      responseBody
    ) {
      try {
        responseBody = JSON.parse(responseBody);
      } catch (ex) {
        // pass
      }
    }
  }

  if (RESPONSE_HANDLERS[response.status]) {
    RESPONSE_HANDLERS[response.status]?.forEach((handler) =>
      handler({
        url,
        method,
        json,
        params,
        headers,
        options,
        response,
        body: responseBody,
      } as ResponseHandlerParams),
    );
  }

  if (RESPONSE_HANDLERS.all) {
    RESPONSE_HANDLERS.all.forEach((handler) => {
      handler({
        url,
        method,
        json,
        params,
        headers,
        options,
        response,
        body: responseBody,
      } as ResponseHandlerParams);
    });
  }

  return {
    status: response.status,
    body: responseBody,
    ok: response.status >= 200 && response.status < 300,
  };
}
