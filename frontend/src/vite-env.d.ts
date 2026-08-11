/// <reference types="vite/client" />

// Vite ?raw import 声明
declare module '*?raw' {
  const value: string;
  export default value;
}
