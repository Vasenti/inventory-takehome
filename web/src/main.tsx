import React, { useEffect, useMemo, useState } from "react";
import ReactDOM from "react-dom/client";
import "./styles.css";

type ProductStock = {
  sku: string;
  name: string;
  quantity: number;
};

type Movement = {
  event_id: string;
  sku: string;
  type: "IN" | "OUT";
  quantity: number;
  occurred_at: string;
};

const apiBaseURL = import.meta.env.VITE_API_BASE_URL ?? "http://localhost:8080";

function App() {
  const [products, setProducts] = useState<ProductStock[]>([]);
  const [selectedSKU, setSelectedSKU] = useState<string>("");
  const [movements, setMovements] = useState<Movement[]>([]);
  const [loadingProducts, setLoadingProducts] = useState(true);
  const [loadingMovements, setLoadingMovements] = useState(false);
  const [error, setError] = useState<string>("");

  useEffect(() => {
    async function loadProducts() {
      try {
        setError("");
        setLoadingProducts(true);
        const response = await fetch(`${apiBaseURL}/products/stock`);
        if (!response.ok) {
          throw new Error("Could not load product stock");
        }

        const data = (await response.json()) as ProductStock[];
        setProducts(data);
        setSelectedSKU(data[0]?.sku ?? "");
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unexpected error");
      } finally {
        setLoadingProducts(false);
      }
    }

    void loadProducts();
  }, []);

  useEffect(() => {
    if (!selectedSKU) {
      setMovements([]);
      return;
    }

    async function loadMovements() {
      try {
        setError("");
        setLoadingMovements(true);
        const response = await fetch(`${apiBaseURL}/products/${selectedSKU}/movements`);
        if (!response.ok) {
          throw new Error("Could not load product movements");
        }

        const data = (await response.json()) as Movement[];
        setMovements(data);
      } catch (err) {
        setError(err instanceof Error ? err.message : "Unexpected error");
      } finally {
        setLoadingMovements(false);
      }
    }

    void loadMovements();
  }, [selectedSKU]);

  const selectedProduct = useMemo(
    () => products.find((product) => product.sku === selectedSKU),
    [products, selectedSKU],
  );

  return (
    <main className="app-shell">
      <section className="inventory-panel">
        <header className="toolbar">
          <div>
            <h1>Inventory</h1>
            <p>{products.length} products</p>
          </div>
          <span className="api-pill">API {apiBaseURL}</span>
        </header>

        {error && <div className="error-banner">{error}</div>}

        <div className="content-grid">
          <section className="product-list" aria-label="Product stock">
            <div className="section-heading">
              <h2>Current stock</h2>
              {loadingProducts && <span>Loading</span>}
            </div>
            <div className="product-table">
              {!loadingProducts && products.length === 0 ? (
                <p className="empty-state">No data available.</p>
              ) : (
                products.map((product) => (
                  <button
                    className={product.sku === selectedSKU ? "product-row selected" : "product-row"}
                    key={product.sku}
                    onClick={() => setSelectedSKU(product.sku)}
                    type="button"
                  >
                    <span>
                      <strong>{product.sku}</strong>
                      <small>{product.name}</small>
                    </span>
                    <b>{product.quantity}</b>
                  </button>
                ))
              )}
            </div>
          </section>

          <section className="movement-list" aria-label="Product movements">
            <div className="section-heading">
              <h2>{selectedProduct ? selectedProduct.name : "Movements"}</h2>
              {loadingMovements ? <span>Loading</span> : <span>{movements.length} events</span>}
            </div>

            <div className="movement-table">
              <div className="movement-header">
                <span>Time</span>
                <span>Type</span>
                <span>Qty</span>
                <span>Event</span>
              </div>
              {movements.map((movement) => (
                <div className="movement-row" key={movement.event_id}>
                  <span>{formatDate(movement.occurred_at)}</span>
                  <span className={movement.type === "IN" ? "type-in" : "type-out"}>{movement.type}</span>
                  <span>{movement.quantity}</span>
                  <code>{movement.event_id}</code>
                </div>
              ))}
              {!loadingMovements && selectedSKU && movements.length === 0 && (
                <p className="empty-state">No data available.</p>
              )}
            </div>
          </section>
        </div>
      </section>
    </main>
  );
}

function formatDate(value: string) {
  return new Intl.DateTimeFormat("en", {
    dateStyle: "medium",
    timeStyle: "short",
  }).format(new Date(value));
}

ReactDOM.createRoot(document.getElementById("root")!).render(
  <React.StrictMode>
    <App />
  </React.StrictMode>,
);
