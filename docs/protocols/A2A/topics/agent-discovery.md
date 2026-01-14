# Descubrimiento de agentes

Kairos soporta tres patrones de discovery sin imponer un mecanismo único:
configuración local, well-known y registry externo.

## Configuración local

Adecuado para entornos corporativos. Define endpoints conocidos y controlados.

## Well-known

A partir de una URL base, se obtiene el Agent Card en la ruta estandarizada.

## Registry externo

Permite discovery dinámico sin ser parte del protocolo A2A. Es opt-in y se
puede reemplazar por el mecanismo corporativo existente.
