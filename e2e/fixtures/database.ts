const BASE_URL = process.env.BASE_URL || 'http://127.0.0.1:5000';

/**
 * Reset the database to a clean state using pgundolog.undo()
 */
export async function resetDatabase(): Promise<void> {
  const response = await fetch(`${BASE_URL}/api/test/reset-db`, {
    method: 'POST',
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Failed to reset database: ${response.status} ${text}`);
  }
}

/**
 * Execute a SQL query and return the results
 * @param sql SQL query string
 * @param params Optional query parameters
 * @returns Array of row objects
 */
export async function queryDatabase(
  sql: string,
  params?: any[]
): Promise<any[]> {
  const response = await fetch(`${BASE_URL}/api/test/query`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify({ sql, params: params || [] }),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Query failed: ${response.status} ${text}`);
  }

  return response.json();
}

/**
 * Create a test user
 * @param attrs User attributes (name, email, password, etc.)
 * @returns Created user object
 */
export async function createUser(attrs: {
  name?: string;
  email?: string;
  password?: string;
  [key: string]: any;
}): Promise<any> {
  const response = await fetch(`${BASE_URL}/api/test/users`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(attrs),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Failed to create user: ${response.status} ${text}`);
  }

  return response.json();
}

/**
 * Create a test feed
 * @param attrs Feed attributes (name, url, etc.)
 * @returns Created feed object
 */
export async function createFeed(attrs?: {
  name?: string;
  url?: string;
  [key: string]: any;
}): Promise<any> {
  const response = await fetch(`${BASE_URL}/api/test/feeds`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(attrs || {}),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Failed to create feed: ${response.status} ${text}`);
  }

  return response.json();
}

/**
 * Create a test item
 * @param attrs Item attributes (feed_id, title, url, etc.)
 * @returns Created item object
 */
export async function createItem(attrs: {
  feed_id?: number;
  title?: string;
  url?: string;
  [key: string]: any;
}): Promise<any> {
  const response = await fetch(`${BASE_URL}/api/test/items`, {
    method: 'POST',
    headers: { 'Content-Type': 'application/json' },
    body: JSON.stringify(attrs),
  });

  if (!response.ok) {
    const text = await response.text();
    throw new Error(`Failed to create item: ${response.status} ${text}`);
  }

  return response.json();
}
