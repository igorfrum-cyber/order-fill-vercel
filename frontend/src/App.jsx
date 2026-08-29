import { useState } from "react";
import { NorthApp } from "./ui/north/NorthApp.jsx";
import { OrderFillApp } from "./ui/order/OrderFillApp.jsx";

export default function App() {
  const [mode, setMode] = useState("order");
  if (mode === "north") return <NorthApp mode={mode} onMode={setMode} />;
  return <OrderFillApp mode={mode} onMode={setMode} />;
}
