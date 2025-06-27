Sample API - Aplicação de Exemplo
Uma API web simples em Go, criada como uma aplicação de exemplo para testar e validar fluxos de deploy automatizados com a ferramenta OpsMaster e o Argo CD.

🎯 Propósito
Este repositório serve como um alvo de deploy para a ferramenta CLI opsmaster. O objetivo é simular um microserviço real em um ambiente de GitOps, demonstrando um fluxo de trabalho completo desde a construção da imagem até a implantação no Kubernetes.

✨ Funcionalidades
API Simples: Um único endpoint (/) que retorna uma mensagem JSON com o nome e a versão do serviço.

Containerizada: Inclui um Dockerfile de múltiplos estágios para a criação de uma imagem Docker otimizada, pequena e segura.

Pronta para GitOps: Contém uma estrutura de chart/ projetada para ser usada com o Argo CD:

Chart.yaml: Define uma dependência de um chart Helm central, promovendo a reutilização e a padronização.

values-{ambiente}.yaml: Arquivos de valores para configurar a aplicação em diferentes ambientes (ex: staging, produção).

🚀 Como Fazer o Deploy
A implantação desta aplicação é orquestrada pela ferramenta opsmaster. O fluxo geral é:

Construir e Publicar a Imagem:

Construa a imagem Docker a partir do Dockerfile neste repositório.

Faça o push da imagem para um registro de contêineres (como Docker Hub, GHCR, etc.) com uma tag de versão (ex: v1.0.0).

Configurar o Argo CD (se for a primeira vez):

Adicione este repositório ao Argo CD:

opsmaster argocd repo add https://github.com/seu-usuario/sample-api.git ...

Crie o projeto no Argo CD:

opsmaster argocd project create meu-projeto ...

Executar o Deploy com OpsMaster:

Use o comando app create para implantar a aplicação, especificando a tag da imagem que você acabou de publicar:

opsmaster argocd app create \
    --app-name "sample-api-stg" \
    --repo-url "https://github.com/seu-usuario/sample-api.git" \
    --repo-path "chart" \
    --values "values-stg.yaml" \
    --set-image-tag "v1.0.0" \
    ... (outras flags de conexão)
