import { render, screen } from "@testing-library/react";
import { expect, test } from "vitest";
import { UploadStage } from "./SetupUpload.jsx";

test("standalone recalculation needs the source and no supplier blank", () => {
 render(<UploadStage recalculate sourceFile={{name:"source.xlsx"}} blankFiles={{}} onSource={()=>{}} onWarehouse={()=>{}} onBlank={()=>{}} onHome={()=>{}} onProcess={()=>{}} />);
 expect(screen.getByRole("button", {name:"Обработать"})).toBeEnabled();
 expect(screen.queryByLabelText("Бланк поставщика")).not.toBeInTheDocument();
 expect(screen.getByText("Обновим рекомендации, сохраним заказ по факту и комментарии.")).toBeInTheDocument();
});
