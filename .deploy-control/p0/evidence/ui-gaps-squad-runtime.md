# Análise de Lacunas de UI: Squad em Seletor de Assignee e Botão de Deletar Runtime (ORQ-18)

- **Autor:** Agy-P0-A8 (wB:p2)
- **Data UTC:** 2026-07-27T11:00:02Z
- **Escritor Único de Código:** Codex56-TL (w5:pC)
- **Modo:** SOMENTE LEITURA / ANÁLISE (Nenhuma alteração em código aplicada)

---

## 1. Seletor de Assignee & Atribuição de Squad

### Localização no Frontend
- **Arquivo:** `packages/views/issues/components/pickers/assignee-picker.tsx`
- **Linhas e Snippet Literal:**
  - Linha 76: `const { data: squads = [] } = useQuery(squadListOptions(wsId));`
  - Linhas 101–103:
    ```tsx
    const filteredSquads = squads
      .filter((s) => !s.archived_at && (s.name.toLowerCase().includes(query) || matchesPinyin(s.name, query)))
      .sort((a, b) => getFreq("squad", b.id) - getFreq("squad", a.id));
    ```
  - Linhas 215–234:
    ```tsx
    {filteredSquads.length > 0 && (
      <PickerSection label={t(($) => $.pickers.assignee.squads_group)}>
        {filteredSquads.map((s) => (
          <PickerItem
            key={s.id}
            selected={isSelected("squad", s.id)}
            onClick={() => {
              onUpdate({
                assignee_type: "squad",
                assignee_id: s.id,
              });
              setOpen(false);
            }}
          >
            <ActorAvatar actorType="squad" actorId={s.id} size={18} />
            <span className="truncate">{s.name}</span>
          </PickerItem>
        ))}
      </PickerSection>
    )}
    ```

### Análise
No Web (`packages/views`), o componente `AssigneePicker` **já possui** a seção de `squads` mapeada e envia `assignee_type: "squad"`. No entanto, em `apps/mobile` (`apps/mobile/components/issue/pickers/assignee-picker-body.tsx`) ou em filtros secundários de busca/filtros de issue, a opção de squad por vezes é ignorada ou tratada separadamente.

---

## 2. Verificação do Campo `project.lead_type`

- **Fato Medido (Banco & Código Backend/Frontend):**
  - **Schema DB (`server/migrations/034_projects.up.sql`, Linha 10):**
    ```sql
    lead_type TEXT CHECK (lead_type IN ('member', 'agent'))
    ```
  - **Tipo Core Frontend (`packages/core/types/project.ts`, Linhas 13 & 28):**
    ```typescript
    lead_type: "member" | "agent" | null;
    ```
- **Conclusão:** `project.lead_type` **NÃO aceita `squad`**. A constraint do PostgreSQL restringe estritamente a `'member'` e `'agent'`. Se for necessário permitir squad como líder de projeto, a migration SQL precisará ser alterada.

---

## 3. Botão de Deletar Runtime

### Localização no Frontend
1. **Componente de Linha da Tabela / Kebab Menu:**
   - **Arquivo:** `packages/views/runtimes/components/runtime-list.tsx`
   - **Linhas 582–584 (Cálculo de Permissão):**
     ```typescript
     canDelete:
       !isPendingCustomRuntime(runtime) &&
       (isAdmin || (!!user && runtime.owner_id === user.id)),
     ```
   - **Linhas 473–475 (Ocultamento do Menu):**
     ```tsx
     if (!canDelete) {
       return <span aria-hidden />;
     }
     ```
2. **Página de Detalhes do Runtime:**
   - **Arquivo:** `packages/views/runtimes/components/runtime-detail.tsx`
   - **Linhas 114, 485–503:**
     ```tsx
     const canDelete = isAdmin || isRuntimeOwner;
     ...
     {canDelete && (
       <div className="border-t pt-3">
         <Button
           variant="ghost"
           size="sm"
           className="h-8 w-full justify-start gap-2 text-destructive hover:bg-destructive/10 hover:text-destructive"
           onClick={onDelete}
         >
           <Trash2 className="h-3.5 w-3.5" />
           {t(($) => $.detail.delete_button)}
         </Button>
       </div>
     )}
     ```

### Proposta de Mudança Mínima (para Codex56-TL)
1. **Exposição do Botão de Exclusão:**
   Se a API `DELETE /api/runtimes/{id}` aceita deleção por qualquer membro autorizado da workspace (ou se a regra do backend for ajustada), em `runtime-list.tsx:582` e `runtime-detail.tsx:114`, afrouxar o guard da UI para permitir o botão `Delete` ou exibir a ação desabilitada com tooltip descritivo em vez de omitir o botão totalmente.
2. **Exibir Ação no Card/Header:**
   No card principal da lista de máquinas em `runtimes-page.tsx:794`, adicionar a prop de ação rápida de deleção invocando a rota `DELETE /api/runtimes/{id}`.
