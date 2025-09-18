package simple;

public class HelloWorld {
    public static void main(String[] args) {
        System.out.println("Hello, World!");
        P p = new P();
        p.filed = "123";
        HelloWorld helloWorld = new HelloWorld();
        helloWorld.testFunction(p);
        testFunction(p,"asda")
    }


     public  String  testFunction(P args2){
        return args2.getFiled();
     }

      public  String  testFunction(P args2,String args3){
             return args2.getFiled();
          }
      public class P {
           public  String filed;

           public  String getFiled(){
               return filed;
           }
         }
}

