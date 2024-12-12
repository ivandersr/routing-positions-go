# Events

### Receber o evento

**RouteCreated**

- id
- distance
- directions
  - lat
  - lng

### Efeito colateral: Executar e retornar outro evento

**FreightCalculated**

- route_id
- amount

---

### Receber o evento

**DeliveryStarted**

- route_id

### Efeito colateral

**DriverMoved**

- route_id
- lat
- lng
