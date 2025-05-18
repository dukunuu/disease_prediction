import { clsx, type ClassValue } from "clsx"
import { twMerge } from "tailwind-merge"

export function cn(...inputs: ClassValue[]) {
  return twMerge(clsx(inputs))
}

export function decodeBase64Utf8(base64String: string): string | null {
  try {
    // 1. Decode Base64 to a "binary" string (each character code represents a byte)
    const binaryString = atob(base64String);

    // 2. Convert the binary string into a Uint8Array (an array of actual bytes)
    const len = binaryString.length;
    const bytes = new Uint8Array(len);
    for (let i = 0; i < len; i++) {
      bytes[i] = binaryString.charCodeAt(i);
    }

    // 3. Use TextDecoder to decode the bytes as UTF-8
    const decoder = new TextDecoder('utf-8'); // Specify UTF-8 encoding
    return decoder.decode(bytes);

  } catch (error) {
    // Handle potential errors from atob() or TextDecoder
    console.error("Error decoding Base64/UTF-8 string:", error);
    return null; // Return null to indicate failure
  }
}
