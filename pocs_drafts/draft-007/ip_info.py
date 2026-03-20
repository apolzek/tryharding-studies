import os
import urllib.request as urllib2
import json

def get_ip_info():
    """
    Obtém informações detalhadas sobre um endereço IP usando a API ip-api.com
    """
    while True:
        try:
            # Solicita o IP do usuário
            ip = input("What is your target IP: ")
            
            # Valida se o IP não está vazio
            if not ip.strip():
                print("Por favor, insira um endereço IP válido.")
                continue
            
            # URL da API
            url = "http://ip-api.com/json/" + ip
            
            # Faz a requisição
            response = urllib2.urlopen(url)
            data = response.read()
            
            # Decodifica o JSON
            values = json.loads(data)
            
            # Verifica se a consulta foi bem-sucedida
            if values.get("status") == "fail":
                print(f"Erro: {values.get('message', 'IP inválido')}")
                continue
            
            # Exibe as informações
            print(f"IP: {values.get('query', 'N/A')}")
            print(f"City: {values.get('city', 'N/A')}")
            print(f"ISP: {values.get('isp', 'N/A')}")
            print(f"Country: {values.get('country', 'N/A')}")
            print(f"Region: {values.get('region', 'N/A')}")
            print(f"Timezone: {values.get('timezone', 'N/A')}")
            
            # Informações adicionais que a API fornece
            print(f"Region Name: {values.get('regionName', 'N/A')}")
            print(f"ZIP Code: {values.get('zip', 'N/A')}")
            print(f"Latitude: {values.get('lat', 'N/A')}")
            print(f"Longitude: {values.get('lon', 'N/A')}")
            print(f"Organization: {values.get('org', 'N/A')}")
            
            break
            
        except urllib.error.URLError as e:
            print(f"Erro de conexão: {e}")
            print("Verifique sua conexão com a internet.")
            break
        except json.JSONDecodeError:
            print("Erro ao processar a resposta da API.")
            break
        except KeyboardInterrupt:
            print("\nPrograma interrompido pelo usuário.")
            break
        except Exception as e:
            print(f"Erro inesperado: {e}")
            break

if __name__ == "__main__":
    print("=== Ferramenta de Informações de IP ===")
    get_ip_info()