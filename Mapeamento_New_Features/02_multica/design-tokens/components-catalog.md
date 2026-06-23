# Design System — Catálogo de Componentes (Multica.ai)

Este documento descreve os padrões visuais e a especificação técnica dos componentes do Design System do **Multica.ai**, baseado no framework **Tailwind CSS** e nos padrões do **shadcn/ui**.

---

## 1. Botões (Buttons)

Botões usam uma transição suave e são construídos com foco em usabilidade e feedback tátil.

* **Bordas**: `border-radius: var(--radius)` (padrão `0.625rem` / `10px`).
* **Tipografia**: `font-weight: 500` (Medium), `font-size: var(--text-sm)` (`0.875rem`).
* **Espaçamento padrão**: `py-2 px-4` (vertical/horizontal).
* **Efeitos**: Transição de cor suave (`transition-colors`).

### Variantes de Estilo

1. **Primary Button**
   * **Light Mode**: Fundo em oklch preto/escuro (`var(--primary)`), texto claro (`var(--primary-foreground)`).
   * **Dark Mode**: Fundo claro/branco (`var(--primary)`), texto escuro (`var(--primary-foreground)`).
   * **Estilos**: `bg-primary text-primary-foreground hover:bg-primary/90`.

2. **Secondary / Outline Button**
   * **Estilos**: `border border-input bg-background hover:bg-accent hover:text-accent-foreground`.
   * **Bordas**: OKLCH de borda/input cinza-claro.

3. **Ghost Button**
   * **Estilos**: `hover:bg-accent hover:text-accent-foreground text-foreground bg-transparent`.
   * Usado para botões utilitários no cabeçalho e interações secundárias.

4. **Destructive Button**
   * **Estilos**: `bg-destructive text-destructive-foreground hover:bg-destructive/90`.
   * Usado para ações de exclusão ou cancelamento crítico.

---

## 2. Campos de Entrada (Inputs)

Campos de formulário possuem layouts consistentes com foco e validação integrados.

* **Text Input & Textarea**:
  * **Estilos**: `flex h-9 w-full rounded-md border border-input bg-transparent px-3 py-1 text-sm shadow-sm transition-colors file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-muted-foreground focus-visible:outline-none focus-visible:ring-1 focus-visible:ring-ring disabled:cursor-not-allowed disabled:opacity-50`.
  * **Borda**: `var(--border)` (Light: `oklch(92% .004 286.32)` / Dark: `oklch(100% 0 0/.1)`).
  * **Placeholder**: `var(--muted-foreground)` (Light: `oklch(55.2% .016 285.938)` / Dark: `oklch(70.5% .015 286.067)`).
  * **Focus Ring**: `var(--ring)` com espessura de `1px` azul/roxo.

---

## 3. Badges e Tags

Usados para status, prioridade e marcações gerais.

* **Estilo Base**: `inline-flex items-center rounded-full border px-2.5 py-0.5 text-xs font-semibold transition-colors focus:outline-none focus:ring-2 focus:ring-ring focus:ring-offset-2`.
* **Badges Semânticos**:
  * **Success Tag**: Fundo verde-claro, texto verde-escuro (`bg-success/10 text-success`).
  * **Warning Tag**: Fundo amarelo/laranja-claro, texto escuro (`bg-warning/10 text-warning`).
  * **Info Tag**: Fundo azul-claro, texto azul-escuro (`bg-info/10 text-info`).
  * **Destructive Tag**: Fundo vermelho-claro, texto vermelho-escuro (`bg-destructive/10 text-destructive`).

---

## 4. Avatares (Avatars)

Avatares no Multica possuem diferenciação visual clara entre humanos e agentes.

* **Human Avatar**:
  * Formato circular perfeito.
  * Iniciais do usuário em caixa alta centralizadas, com fundo gerado dinamicamente com base nas cores da marca.
* **Agent Avatar (Coding Bots)**:
  * Formato circular perfeito.
  * Ícone interno de robô/bot com uma borda azul sutil (`border border-info/20`).
  * **Badge de Agente**: Exibe o status da atividade do robô com badges flutuantes coloridas (verde para online/trabalhando, azul para ocioso).
* **Tamanhos padrão**:
  * Pequeno (Sidebar/Dropdown): `18px` ou `20px` de diâmetro.
  * Médio (Feed/Timeline): `24px` ou `32px` de diâmetro.
  * Grande (Páginas de Perfil/Config): `40px` ou `48px` de diâmetro.

---

## 5. Skeletons (Carregamento Dinâmico)

Padrão de carregamento visual ("Loading Skeleton") usado em toda a plataforma para mitigar a latência.

* **Estilos**: `animate-pulse rounded-md bg-muted`.
* **Animação**: Opacidade oscilando continuamente (`pulse`) para dar a sensação de progresso ativo.
* **Cor de Fundo**: Cinza-claro suave em Light mode e cinza-escuro em Dark mode (`var(--muted)`).

---

## 6. Sockets e Live Activity (WebSocket Feed)

* Componentes interativos de log que piscam ou exibem indicadores brilhantes de atividade (`animate-ping`) para ações do WebSocket em tempo real.
* Uso de ícones circulares dinâmicos que mostram o progresso incremental da tarefa.
