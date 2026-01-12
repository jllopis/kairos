Dataset: DemoVentasGastos

This dataset contains two sheets represented as CSV files:
- Ventas: sales transactions with region, product, net_sales, and margin.
- Gastos: operating expenses with category, amount, and department.

Definitions:
- Q4 refers to Oct, Nov, Dec.
- net_sales is revenue after discounts and returns.
- margin is gross margin ratio (0-1).

Business rules:
- Use net_sales for revenue totals.
- Group by region or product when ranking performance.
- For anomalies in Gastos, flag amounts that deviate materially from the mean for the current month.
