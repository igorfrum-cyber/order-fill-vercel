export function priceOrderRows(rows,settings) {
  return rows.map(r=>{
    const config=settings[r.group],selected=r.prices?.find(p=>p.id===config?.column);
    const price=selected?.price>0?Math.round(selected.price*(1-config.discount/100)*100)/100:0;
    return {...r,price,pricing:{column:selected?.column,label:selected?.label,basePrice:selected?.price,discount:config?.discount || 0}};
  });
}
