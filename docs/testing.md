# Testes e fixtures de imagem

## Incidente: consumo extremo de memória ao gerar PNG

Em 30/09/2026, a suíte de testes da UI chegou a consumir aproximadamente
10 GB de RAM e ficou travada nos testes de thumbnail de imagem.

### Causa

O helper `writeTestPNG` passava `image.NewUniform(...)` diretamente para
`png.Encode`. `image.NewUniform` não cria uma imagem pequena: ele representa
uma imagem uniforme de tamanho infinito. O encoder PNG tenta percorrer os
limites completos da imagem e pode iniciar uma alocação ou processamento
absurdo.

Isso não era uma incompatibilidade do Fyne nem um vazamento do aplicativo. Era
uma fixture de teste incompatível com um encoder que precisa de limites finitos.

### Regra preventiva

Nunca passar `image.NewUniform` diretamente para encoders de arquivo, como
`png.Encode` ou `jpeg.Encode`.

Para fixtures, sempre usar uma imagem com retângulo finito:

```go
fixture := image.NewRGBA(image.Rect(0, 0, 2, 2))
fixture.Set(0, 0, color.RGBA{R: 40, G: 80, B: 120, A: 255})
if err := png.Encode(file, fixture); err != nil {
	t.Fatalf("encode test image: %v", err)
}
```

`image.NewUniform` continua adequado como fonte de cor em operações de desenho
limitadas por um destino finito, como o ícone 16×16 da bandeja. O risco ocorre
quando o objeto é serializado como uma imagem completa.

### Checklist antes de adicionar fixtures de imagem

- Confirmar que `Bounds().Dx()` e `Bounds().Dy()` são finitos e pequenos.
- Não usar `image.NewUniform` como entrada direta de um encoder.
- Executar primeiro o teste específico da fixture.
- Executar `go test ./...` e `CGO_ENABLED=0 go test -tags ci ./...`.
- Se houver suspeita de alocação, validar o pico com `/usr/bin/time -f 'max RSS: %M KB'`.

Referência: a documentação oficial do Go descreve `image.Uniform` como uma
imagem de tamanho infinito: <https://pkg.go.dev/image>.

## Incidente: histórico com linhas vazias

Uma regressão visual fez o popup informar que havia itens, mas mostrar linhas
pretas sem texto e sem miniatura.

### Causa

As linhas eram um tipo próprio que apenas embutia `*fyne.Container`. Embora os
filhos (`Label` e `canvas.Image`) existissem e tivessem tamanho correto, o
walker de pintura do Fyne reconhece o contêiner concreto `*fyne.Container`, não
um wrapper que o embute. O resultado era uma linha presente no `widget.List`,
mas sem filhos renderizáveis.

### Regra preventiva

Callbacks de `widget.List` devem retornar contêineres Fyne nativos ou widgets
com renderer próprio. Se for necessário manter estado auxiliar para uma linha,
ele deve ser reconstruído a partir do `*fyne.Container` no callback, sem colocar
o wrapper na árvore visual.

Para linhas que alternam filhos visíveis, o layout deve ser executado novamente
antes do refresh. No histórico, a altura de 64 px é aplicada com
`List.SetItemHeight` e a miniatura de 48 px é relayoutada quando aparece.

### Cobertura obrigatória

- Testar `go test ./internal/ui/fyne -run TestPopupCanvasActuallyRendersHistoryContent`;
- validar texto e miniatura no canvas, não apenas os campos `Label.Text` e
  `Image.File`;
- após alterar a lista, recompilar o binário e encerrar qualquer processo
  antigo que ainda mantenha os grabs X11 antes do teste manual;
- confirmar no log e em `~/Imagens/Captures` que uma captura concluída criou um
  PNG, pois a notificação não é a fonte de verdade da persistência.

## Incidente: imagem do clipboard registrada como texto

Uma seleção de imagem no X11 chegou a aparecer no histórico duas vezes: uma
vez como miniatura e outra como uma sequência hexadecimal relacionada ao PNG.

### Causa

O leitor textual anterior chamava `xclip -out -selection clipboard` sem
solicitar um MIME type. Quando a seleção corrente era `image/png` (inclusive
após uma seleção explícita de imagem no histórico), o adaptador podia devolver
metadados ou uma representação auxiliar como se fosse texto. O monitor então
criava uma linha textual indevida.

### Regra preventiva

O adaptador X11 deve consultar `TARGETS` antes de ler o clipboard. Se qualquer
alvo `image/*` estiver presente, o monitor deve ler a imagem, salvá-la como um
arquivo estável em `~/Imagens/Captures` e registrá-la como imagem no histórico.
A leitura textual direta ainda deve retornar `ErrClipboardNotText`, para nunca
converter a imagem ou seus metadados em texto. Só uma seleção explícita de uma
linha no histórico pode escrever novamente no clipboard do sistema.

Também não se deve chamar `WriteText` ou `WriteImage` ao apenas sincronizar o
clipboard ou ao abrir o popup. Essas operações somente registram o item. A
conclusão de uma captura é uma ação explícita do usuário e deve publicar o PNG
recém-criado no clipboard; a seleção de uma linha do histórico também pode
escrever no clipboard.

### Cobertura obrigatória

- testar que `image/png` é rejeitado mesmo se `UTF8_STRING` também estiver em
  `TARGETS` pelo leitor textual;
- testar que `image/png` é salvo e registrado como uma imagem, sem alterar o
  clipboard;
- testar que uma captura recém-concluída é escrita no clipboard e não deixa o
  PNG anterior disponível para a próxima colagem;
- testar que um clipboard textual solicita `UTF8_STRING` e é registrado;
- testar que abrir o popup imediatamente após copiar uma imagem já mostra a
  entrada, sem depender do próximo polling;
- testar que o histórico tem no máximo 10 itens e remove o mais antigo;
- testar visualmente o popup com texto e imagem, incluindo a borda arredondada
  e o fundo com alpha aproximado de 90%;
- depois de selecionar uma imagem, confirmar que ela permanece no clipboard e
  que nenhuma linha hexadecimal nova aparece no histórico nem uma duplicata da
  imagem é criada.

## Incidente: captura colava a imagem anterior

Uma captura feita com `Super+Shift+S` era salva corretamente em
`~/Imagens/Captures` e aparecia no histórico, mas a colagem seguinte ainda
usava a captura anterior.

### Causa

O handler de captura chamava apenas `History.AddImage(path)`. Esse método
registrava o caminho na memória, mas não atualizava a seleção `image/png` do
clipboard X11. O monitor passivo também não deveria fazer essa escrita, pois
ele só observa o clipboard; por isso o sistema continuava oferecendo o PNG
anterior para os aplicativos de destino.

### Regra preventiva

Depois de uma captura concluída, o caso de uso deve publicar o mesmo arquivo
no clipboard por meio de `Manager.RecordCapturedImage`. A operação marca o
eco esperado para que o polling não registre a imagem novamente como uma nova
linha. Se a publicação falhar, a captura ainda deve permanecer no histórico e
o erro deve ser registrado para diagnóstico.

### Cobertura obrigatória

- testar que o caminho recém-capturado é enviado a `WriteImage`;
- testar que a captura aparece no histórico como imagem;
- testar que o primeiro ciclo de sincronização do clipboard não cria uma
  segunda linha para o eco da captura;
- validar manualmente: tirar duas capturas diferentes e colar imediatamente
  após a segunda; o aplicativo de destino deve receber a segunda imagem.

## Limite e estilo do histórico

O limite global do histórico é 10 itens, misturando textos e imagens. A entrada
mais recente fica no topo; ao inserir a 11ª entrada, a mais antiga é removida.
O popup usa uma linha de 64 px, miniatura de 48 px quando aplicável, raio de
10 px e fundo com alpha 230 (aproximadamente 90%).

## Incidente: janela auxiliar aberta pela bandeja

O Fyne abre a janela passada a `SetSystemTrayWindow` quando o ícone da bandeja
é acionado. Essa janela chegou a aparecer vazia ou mostrar uma tela sem ação
útil.

### Regra preventiva

O aplicativo não deve registrar `SetSystemTrayWindow` para a bandeja. O ícone
deve permanecer residente e o menu deve conter somente `Sair`; qualquer ação do
histórico continua acessível pelo atalho `Super+Shift+V`.

### Cobertura obrigatória

- testar que a configuração da bandeja registra menu e ícone;
- testar que nenhuma janela é registrada para abertura pelo ícone;
- validar manualmente no Cinnamon que clicar no ícone não cria uma tela ou
  janela auxiliar e que `Sair` encerra o processo pelo menu.

## Smoke test dos atalhos X11

O atalho do histórico é `Super+Shift+V`. O atalho `Super+Shift+S` continua
reservado para captura de área. Antes do teste manual, deve existir somente uma
instância do aplicativo, porque os grabs globais X11 são exclusivos; uma
instância antiga pode manter o atalho anterior ocupado e produzir um resultado
que não corresponde ao binário recém-compilado.

## Smoke test do pacote instalável

O binário solto em `build/super-v` não é um pacote de sistema e pode perder a
permissão de execução quando o projeto está em uma partição montada com
permissões fixas. A distribuição deve ser gerada como `.deb`:

```bash
make package-test
sudo apt install ./build/super-v_0.1.0_amd64.deb
```

O smoke test deve confirmar que o pacote contém:

- `/usr/bin/super-v` com modo `0755`;
- `/usr/share/applications/super-v.desktop` para o menu;
- `/etc/xdg/autostart/super-v.desktop` para iniciar os atalhos após o login.

Não validar a instalação clicando no binário sem extensão; o gerenciador de
pacotes só deve ser usado para o arquivo `.deb`. O `make package-test` valida o
metadado, a arquitetura e os três caminhos antes da instalação.
