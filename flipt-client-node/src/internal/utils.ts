export function snakeToCamel(str: string): string {
  return str.replace(/([-_][a-z])/g, (group) =>
    group.toUpperCase().replace('-', '').replace('_', '')
  );
}

export function camelToSnake(str: string): string {
  return str.replace(/[A-Z]/g, (letter) => `_${letter.toLowerCase()}`);
}

export function deserialize<T>(data: any): T {
  const result: any = {};
  for (const key in data) {
    if (Object.prototype.hasOwnProperty.call(data, key)) {
      const camelKey = snakeToCamel(key);
      result[camelKey] = data[key];
    }
  }
  return result as T;
}

export function serialize<T>(data: T): any {
  const result: any = {};
  for (const key in data) {
    if (Object.prototype.hasOwnProperty.call(data, key)) {
      const snakeKey = camelToSnake(key);
      result[snakeKey] = (data as any)[key];
    }
  }
  return result;
}
