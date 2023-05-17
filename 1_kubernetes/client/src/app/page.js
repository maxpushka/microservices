export default function Home() {
  const [hellostatus, sethellostatus] = usestate('');
  const [greetstatus, setgreetstatus] = usestate('');

  useeffect(() => {
    const checkservices = async () => {
      try {
        const helloresponse = await fetch('/hello');
        if (helloresponse.ok) {
          sethellostatus('online');
        } else {
          sethellostatus('offline');
        }

        const greetresponse = await fetch('/greet');
        if (greetresponse.ok) {
          setgreetstatus('online');
        } else {
          setgreetstatus('offline');
        }
      } catch (error) {
        sethellostatus('error');
        setgreetstatus('error');
        console.error(error);
      }
    };

    checkservices();
  }, []);

  return (
    <div>
      <h1>service pinger</h1>
      <div>
        <h2>"hello, world!" service status: {hellostatus}</h2>
        <h2>"greet" service status: {greetstatus}</h2>
      </div>
    </div>
  );
}
